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
	defMaxReadingBytesSize = 1 << 10 //1k
	defTimeoutDuration     = 8 * time.Second
)

var ErrOverMaxReadingSize = errors.New("over max reading size")

type Socket struct {
	lock                 *sync.Mutex
	conn                 net.Conn
	maxReadingBytesSize  uint32
	maxBufferSizeInPool  uint64
	bufferPool           *hpool.Pool
	readTimeoutDuration  time.Duration
	writeTimeoutDuration time.Duration
}

func NewSocket(c net.Conn) *Socket {
	return &Socket{
		lock:                 &sync.Mutex{},
		conn:                 c,
		maxReadingBytesSize:  defMaxReadingBytesSize,
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
	for {
		b, e := s.read(s.getBuffer())
		if e != nil {
			s.putBuffer(b)
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
	b, e := s.read(s.getBuffer())
	if e != nil {
		s.putBuffer(b)
		return nil, e
	}
	return b, nil
}

func (s *Socket) Write(b []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration))

	bf := s.getBuffer()
	defer s.putBuffer(bf)
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

func (s *Socket) getBuffer() *hbuffer.Buffer {
	if s.bufferPool != nil {
		return s.bufferPool.Get().(*hbuffer.Buffer)
	}
	return hbuffer.NewBuffer()
}

func (s *Socket) putBuffer(bf *hbuffer.Buffer) {
	if s.bufferPool != nil {
		s.bufferPool.Put(bf)
	}
}
