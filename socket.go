package hnet

import (
	"net"
	"sync"

	"errors"

	"time"

	"github.com/hqpko/hbuffer"
)

const (
	defMaxReadingBytesSize = 1 << 12 //4k
	defMaxBufferPoolSize   = 1 << 6  //64
	defTimeoutDuration     = 8 * time.Second
)

var ErrOverMaxReadingSize = errors.New("over max reading size")

type Socket struct {
	lock                 *sync.Mutex
	conn                 net.Conn
	poolSize             uint32
	maxReadingBytesSize  uint32
	maxBufferPoolSize    uint64
	bufferPool           chan *hbuffer.Buffer
	readTimeoutDuration  time.Duration
	writeTimeoutDuration time.Duration
}

func createSocket(c net.Conn) *Socket {
	return &Socket{
		lock:                 &sync.Mutex{},
		conn:                 c,
		maxReadingBytesSize:  defMaxReadingBytesSize,
		maxBufferPoolSize:    defMaxBufferPoolSize,
		readTimeoutDuration:  defTimeoutDuration,
		writeTimeoutDuration: defTimeoutDuration,
	}
}

func (s *Socket) SetPoolSize(size uint32) {
	s.poolSize = size
}

func (s *Socket) SetMaxReadingBytesSize(size uint32) {
	s.maxReadingBytesSize = size
}

func (s *Socket) SetMaxPoolBufferSize(size uint64) {
	s.maxBufferPoolSize = size
}

func (s *Socket) SetTimeoutDuration(d time.Duration) {
	s.readTimeoutDuration = d
	s.writeTimeoutDuration = d
}

func (s *Socket) SetReadTimeoutDuration(d time.Duration) {
	s.readTimeoutDuration = d
}

func (s *Socket) SetWriteTimeoutDuration(d time.Duration) {
	s.writeTimeoutDuration = d
}

func (s *Socket) ReadWithCallback(callback func(*hbuffer.Buffer)) error {
	s.initBufferPool()

	for {
		b, e := s.read(s.PutBuffer())
		if e != nil {
			s.PushBuffer(b)
			return e
		}
		callback(b)
	}
}

func (s *Socket) ReadWithChan(c chan *hbuffer.Buffer) error {
	return s.ReadWithCallback(
		func(b *hbuffer.Buffer) {
			c <- b
		},
	)
}

func (s *Socket) ReadOne() (*hbuffer.Buffer, error) {
	s.initBufferPool()

	b, e := s.read(s.PutBuffer())
	if e != nil {
		s.PushBuffer(b)
		return nil, e
	}
	return b, nil
}

func (s *Socket) Write(b []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration))

	bf := s.PutBuffer()
	defer s.PushBuffer(bf)
	bf.WriteUint32(uint32(len(b)))
	_, e := s.conn.Write(bf.GetBytes())
	if e != nil {
		return e
	}
	_, e = s.conn.Write(b)
	return e
}

//WriteWithBuffer default with bytes size
func (s *Socket) WriteWithBuffer(b *hbuffer.Buffer) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration))

	_, e := s.conn.Write(b.GetBytes())
	return e
}

func (s *Socket) read(b *hbuffer.Buffer) (*hbuffer.Buffer, error) {
	s.SetReadDeadline(time.Now().Add(s.readTimeoutDuration))

	_, e := b.ReadFull(s.conn, 4)
	if e != nil {
		return b, e
	}
	l := b.ReadUint32()
	if l > s.maxReadingBytesSize {
		return b, ErrOverMaxReadingSize
	}
	_, e = b.ReadFull(s.conn, uint64(l))
	return b, e
}

func (s *Socket) initBufferPool() {
	if s.bufferPool == nil {
		s.bufferPool = make(chan *hbuffer.Buffer, s.poolSize)
	}
}

func (s *Socket) PutBuffer() *hbuffer.Buffer {
	select {
	case b := <-s.bufferPool:
		return b
	default:
		return hbuffer.NewBuffer()
	}
}

func (s *Socket) PushBuffer(b *hbuffer.Buffer) {
	if b == nil || b.Len() > s.maxBufferPoolSize {
		return
	}
	b.Reset()
	select {
	case s.bufferPool <- b:
	default:
		return
	}
}

func (s *Socket) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *Socket) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *Socket) SetDeadline(t time.Time) {
	s.conn.SetDeadline(t)
}

func (s *Socket) SetReadDeadline(t time.Time) {
	s.conn.SetReadDeadline(t)
}

func (s *Socket) SetWriteDeadline(t time.Time) {
	s.conn.SetWriteDeadline(t)
}

func (s *Socket) Close() {
	s.conn.Close()
}
