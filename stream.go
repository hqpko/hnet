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
)

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
	mainChannel     *hconcurrent.Concurrent
	readCallHandler func(req, resp *hbuffer.Buffer)
}

func NewStream(socket *Socket) *Stream {
	s := &Stream{bufferPool: hutils.NewBufferPool(), socket: socket, pendingLock: sync.Mutex{}, pending: map[uint64]*Call{}}
	return s
}

func (s *Stream) SetReadCallHandler(f func(req, resp *hbuffer.Buffer)) *Stream {
	s.readCallHandler = f
	return s
}

func (s *Stream) NewRequest(oneWay bool) *Call {
	call := &Call{
		buf:    s.bufferPool.Get(),
		oneWay: oneWay,
		c:      make(chan *Call, 1),
	}
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

func (s *Stream) Request(call *Call) *Call {
	// todo, 暂时不加 channel 直接发送，忽略网络传输时间
	if !call.oneWay {
		s.storeCall(call)
	}
	if call.doneIfError(s.send(call.buf)) && !call.oneWay {
		s.deleteCall(call.seq)
	}
	return call
}

func (s *Stream) Response(reqBuffer, respBuffer *hbuffer.Buffer) error {
	s.bufferPool.Put(reqBuffer)
	return s.send(respBuffer)
}

func (s *Stream) send(buffer *hbuffer.Buffer) error {
	defer s.bufferPool.Put(buffer)
	buffer.SetPosition(0)
	buffer.WriteEndianUint32(uint32(buffer.Len() - 4))
	return s.socket.WriteBuffer(buffer)
}

func (s *Stream) Read() error {
	return s.socket.ReadBuffer(func(buffer *hbuffer.Buffer) {
		callType, _ := buffer.ReadByte()
		isRequest := callType&callTypeRequest != 0
		if isRequest {
			oneWay := callType&callTypeOneWay != 0
			s.readRequestHandler(buffer, oneWay)
		} else {
			seq, _ := buffer.ReadUint64()
			if call, ok := s.loadCall(seq); ok {
				call.buf = buffer
				call.done()
			}
		}
	}, s.bufferPool.Get)
}

func (s *Stream) readRequestHandler(buffer *hbuffer.Buffer, oneWay bool) {
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

func (s *Stream) Close() error {
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
