package hnet

import (
	"net"

	"errors"

	"time"

	"github.com/hqpko/hbuffer"
)

var ErrOverMaxReadingSize = errors.New("over max reading size")

type Socket struct {
	net.Conn
	maxReadingBytesSize  uint32
	readTimeoutDuration  time.Duration
	writeTimeoutDuration time.Duration
	handlerGetBuffer     func() *hbuffer.Buffer
	handlerPutBuffer     func(buffer *hbuffer.Buffer)
	cacheBuffer          *hbuffer.Buffer
}

func NewSocket(conn net.Conn, option *Option) *Socket {
	return &Socket{
		Conn:                 conn,
		maxReadingBytesSize:  option.maxReadingByteSize,
		readTimeoutDuration:  option.readTimeoutDuration,
		writeTimeoutDuration: option.writeTimeoutDuration,
		handlerGetBuffer:     option.handlerGetBuffer,
		handlerPutBuffer:     option.handlerPutBuffer,
		cacheBuffer:          hbuffer.NewBuffer(),
	}
}

func (s *Socket) ReadPacket(handlerPacket func(*hbuffer.Buffer)) error {
	for {
		b, e := s.read(s.handlerGetBuffer())
		if e != nil {
			s.handlerPutBuffer(b)
			return e
		}
		handlerPacket(b)
	}
}

func (s *Socket) ReadOnePacket() (*hbuffer.Buffer, error) {
	b, e := s.read(s.handlerGetBuffer())
	if e != nil {
		s.handlerPutBuffer(b)
		return nil, e
	}
	return b, nil
}

func (s *Socket) WritePacket(b []byte) error {
	if e := s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration)); e != nil {
		return e
	}

	s.cacheBuffer.Reset()
	s.cacheBuffer.WriteUint32(uint32(len(b)))
	s.cacheBuffer.WriteBytes(b)
	_, e := s.Write(s.cacheBuffer.GetBytes())
	return e
}

func (s *Socket) read(b *hbuffer.Buffer) (*hbuffer.Buffer, error) {
	if e := s.SetReadDeadline(time.Now().Add(s.readTimeoutDuration)); e != nil {
		return b, e
	}

	_, e := b.ReadFull(s, 4)
	if e != nil {
		return b, e
	}
	l := b.ReadUint32()
	if l > s.maxReadingBytesSize {
		return b, ErrOverMaxReadingSize
	}
	b.Reset()
	_, e = b.ReadFull(s, uint64(l))
	return b, e
}
