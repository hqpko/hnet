package hnet

import (
	"log"
	"sync"
	"sync/atomic"

	"github.com/hqpko/hbuffer"
	"github.com/hqpko/hconcurrent"
	"github.com/hqpko/hutils"
)

const (
	callTypeRequest = 0x01
	callTypeOneWay  = 0x02

	defChannelSize = 1 << 6
)

var callPool = sync.Pool{New: func() interface{} {
	return &Call{buf: hbuffer.NewBuffer(), c: make(chan *Call, 1)}
}}

func getCall() *Call {
	return callPool.Get().(*Call)
}

func putCall(sc *Call) {
	sc.buf.Reset()
	callPool.Put(sc)
}

type Call struct {
	seq    uint64
	oneWay bool
	buf    *hbuffer.Buffer
	err    error
	c      chan *Call
}

func (c *Call) Buffer() *hbuffer.Buffer {
	return c.buf
}

func (c *Call) doneIfError(err error) bool {
	if err != nil {
		c.err = err
		c.done()
		return true
	}
	return false
}

func (c *Call) done() {
	select {
	case c.c <- c:
	default:
		log.Printf("hnet: stream discarding Call reply due to insufficient Done chan capacity")
	}
}

func (c *Call) Done() error {
	<-c.c
	return c.err
}

type Stream struct {
	seq             uint64
	bufferPool      *hutils.BufferPool
	socket          *Socket
	pendingLock     sync.Mutex
	pending         map[uint64]*Call
	mainChannelSize int
	mainChannel     *hconcurrent.Concurrent
	readCallHandler func(req, resp *hbuffer.Buffer)
}

func NewStream(socket *Socket) *Stream {
	s := &Stream{bufferPool: hutils.NewBufferPool(), socket: socket, pendingLock: sync.Mutex{}, pending: map[uint64]*Call{}, mainChannelSize: defChannelSize}
	return s
}

func (s *Stream) SetReadCallHandler(f func(req, resp *hbuffer.Buffer)) *Stream {
	s.readCallHandler = f
	return s
}

func (s *Stream) SetChannelSize(channelSize int) *Stream {
	s.mainChannelSize = channelSize
	return s
}

func (s *Stream) Run() error {
	s.mainChannel = hconcurrent.NewConcurrent(s.mainChannelSize, 1, s.handlerChannel)
	s.mainChannel.Start()

	return s.socket.ReadBuffer(func(buffer *hbuffer.Buffer) {
		callType, _ := buffer.ReadByte()
		isRequest := callType&callTypeRequest != 0
		if isRequest {
			oneWay := callType&callTypeOneWay != 0
			s.handlerRequest(buffer, oneWay)
		} else {
			s.handlerResponse(buffer)
		}
	}, s.bufferPool.Get)
}

func (s *Stream) handlerChannel(i interface{}) interface{} {
	if call, ok := i.(*Call); ok { // request call
		if !call.oneWay {
			s.storeCall(call)
		}
		if call.doneIfError(s.send(call.buf)) && !call.oneWay {
			s.deleteCall(call.seq)
		}
	} else if respBuffer, ok := i.(*hbuffer.Buffer); ok {
		if err := s.send(respBuffer); err != nil {
			log.Printf("hrpc: stream send response error:%s", err.Error())
		}
	}
	return nil
}

func (s *Stream) handlerRequest(buffer *hbuffer.Buffer, oneWay bool) {
	var respBuffer *hbuffer.Buffer
	if !oneWay {
		respBuffer = s.bufferPool.Get()
		respBuffer.WriteEndianUint32(0) // space for len
		respBuffer.WriteByte(0)
		seq, _ := buffer.ReadUint64()
		respBuffer.WriteUint64(seq)
	}
	s.readCallHandler(buffer, respBuffer)
}

func (s *Stream) handlerResponse(buffer *hbuffer.Buffer) {
	seq, _ := buffer.ReadUint64()
	if call, ok := s.loadCall(seq); ok {
		call.buf = buffer
		call.done()
	} else {
		log.Printf("hrpc: stream handler response, no call with seq:%d", seq)
	}
}

func (s *Stream) NewCall(oneWay bool) *Call {
	call := getCall()
	call.oneWay = oneWay
	call.buf.WriteEndianUint32(0) // space for len
	if !oneWay {
		call.buf.WriteByte(callTypeRequest)
		call.seq = atomic.AddUint64(&s.seq, 1)
		call.buf.WriteUint64(call.seq)
	} else {
		call.buf.WriteByte(callTypeOneWay | callTypeRequest)
	}
	return call
}

func (s *Stream) Call(call *Call) {
	s.mainChannel.MustInput(call)
}

func (s *Stream) Reply(reqBuffer, respBuffer *hbuffer.Buffer) {
	s.bufferPool.Put(reqBuffer)
	s.mainChannel.MustInput(respBuffer)
}

func (s *Stream) send(buffer *hbuffer.Buffer) error {
	defer s.bufferPool.Put(buffer)
	buffer.SetPosition(0)
	buffer.WriteEndianUint32(uint32(buffer.Len() - 4))
	return s.socket.WriteBuffer(buffer)
}

func (s *Stream) Close() error {
	s.mainChannel.Stop()
	return s.socket.Close()
}

func (s *Stream) storeCall(call *Call) {
	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	s.pending[call.seq] = call
}

func (s *Stream) deleteCall(seq uint64) {
	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	delete(s.pending, seq)
}

func (s *Stream) loadCall(seq uint64) (*Call, bool) {
	s.pendingLock.Lock()
	defer s.pendingLock.Unlock()
	if call, ok := s.pending[seq]; ok {
		delete(s.pending, seq)
		return call, ok
	}
	return nil, false
}
