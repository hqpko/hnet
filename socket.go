package hnet

import (
	"net"
	"sync"

	"errors"

	"time"

	"github.com/hqpko/hbuffer"
	"github.com/hqpko/hpool"
)

const (
	defBufferPoolSize      = 1 << 2  //4
	defMaxReadingBytesSize = 1 << 10 //1k
	defTimeoutDuration     = 8 * time.Second
)

var ErrOverMaxReadingSize = errors.New("over max reading size")

type Socket struct {
	lock                 *sync.Mutex
	conn                 net.Conn
	bufferPoolSize       int
	maxReadingBytesSize  uint32
	maxBufferSizeInPool  uint64
	bufferPool           *hpool.Pool
	readTimeoutDuration  time.Duration
	writeTimeoutDuration time.Duration

	removeOverMaxSizeBufferCount uint32
}

func NewSocket(c net.Conn) *Socket {
	return &Socket{
		lock:                 &sync.Mutex{},
		conn:                 c,
		maxReadingBytesSize:  defMaxReadingBytesSize,
		bufferPoolSize:       defBufferPoolSize,
		readTimeoutDuration:  defTimeoutDuration,
		writeTimeoutDuration: defTimeoutDuration,
	}
}

func (s *Socket) SetBufferPool(p *hpool.Pool) {
	s.bufferPool = p
}

func (s *Socket) SetMaxReadingBytesSize(size uint32) {
	s.maxReadingBytesSize = size
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
		b, e := s.read(s.bufferPool.Get().(*hbuffer.Buffer))
		if e != nil {
			s.bufferPool.Put(b)
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

	b, e := s.read(s.bufferPool.Get().(*hbuffer.Buffer))
	if e != nil {
		s.bufferPool.Put(b)
		return nil, e
	}
	return b, nil
}

func (s *Socket) Write(b []byte) error {
	s.initBufferPool()
	s.lock.Lock()
	defer s.lock.Unlock()

	s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration))

	bf := s.bufferPool.Get().(*hbuffer.Buffer)
	defer s.bufferPool.Put(bf)
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
	s.initBufferPool()
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
		s.bufferPool = hpool.NewPool(func() interface{} {
			return hbuffer.NewBuffer()
		}, s.bufferPoolSize)
	}
}

func (s *Socket) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *Socket) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

func (s *Socket) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

func (s *Socket) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

func (s *Socket) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}

func (s *Socket) Close() error {
	return s.conn.Close()
}
