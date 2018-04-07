package hnet

import (
	"net"
	"sync"

	"errors"

	"time"

	"sync/atomic"

	"github.com/hqpko/hbuffer"
	"github.com/hqpko/hpool"
)

const (
	defMaxReadingBytesSize = 1 << 12 //4k
	defMaxBufferSizeInPool = 1 << 12 //4k
	defBufferPoolSize      = 1 << 6  //64
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

	debug                        bool
	removeOverMaxSizeBufferCount uint32
}

func createSocket(c net.Conn) *Socket {
	return &Socket{
		lock:                 &sync.Mutex{},
		conn:                 c,
		maxReadingBytesSize:  defMaxReadingBytesSize,
		bufferPoolSize:       defBufferPoolSize,
		maxBufferSizeInPool:  defMaxBufferSizeInPool,
		readTimeoutDuration:  defTimeoutDuration,
		writeTimeoutDuration: defTimeoutDuration,
	}
}

func (s *Socket) SetDebug(b bool) {
	s.debug = b
	if s.bufferPool != nil {
		s.bufferPool.SetDebug(b)
	}
}

func (s *Socket) SetBufferPoolSize(size int) {
	s.bufferPoolSize = size
}

func (s *Socket) SetMaxReadingBytesSize(size uint32) {
	s.maxReadingBytesSize = size
}

func (s *Socket) SetMaxBufferSizeInPool(size uint64) {
	s.maxBufferSizeInPool = size
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
		b, e := s.read(s.GetBuffer())
		if e != nil {
			s.PutBuffer(b)
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

	b, e := s.read(s.GetBuffer())
	if e != nil {
		s.PutBuffer(b)
		return nil, e
	}
	return b, nil
}

func (s *Socket) Write(b []byte) error {
	s.initBufferPool()
	s.lock.Lock()
	defer s.lock.Unlock()

	s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration))

	bf := s.GetBuffer()
	defer s.PutBuffer(bf)
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
		s.bufferPool.SetDebug(s.debug)
	}
}

func (s *Socket) GetBuffer() *hbuffer.Buffer {
	return s.bufferPool.Get().(*hbuffer.Buffer)
}

func (s *Socket) PutBuffer(b *hbuffer.Buffer) {
	if b == nil {
		return
	}
	if b.Len() > s.maxBufferSizeInPool {
		if s.debug {
			atomic.AddUint32(&s.removeOverMaxSizeBufferCount, 1)
		}
		return
	}
	b.Reset()
	s.bufferPool.Put(b)
}

func (s *Socket) GetBufferPoolDebugInfo() (uint32, uint32, uint32, uint32, uint32) {
	newCount, getCount, putCount, removeCount := s.bufferPool.GetDebugInfo()
	return newCount, getCount, putCount, removeCount, s.removeOverMaxSizeBufferCount
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
