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
	readBuffer           *hbuffer.Buffer
	writeBuffer          *hbuffer.Buffer
	handlerGetBuffer     func() *hbuffer.Buffer
	handlerPutBuffer     func(buffer *hbuffer.Buffer)
}

func NewSocket(conn net.Conn, option *Option) *Socket {
	return &Socket{
		Conn:                 conn,
		maxReadingBytesSize:  option.maxReadingByteSize,
		readTimeoutDuration:  option.readTimeoutDuration,
		writeTimeoutDuration: option.writeTimeoutDuration,
		readBuffer:           hbuffer.NewBuffer(),
		writeBuffer:          hbuffer.NewBuffer(),
		// handlerGetBuffer:     option.handlerGetBuffer,
		// handlerPutBuffer:     option.handlerPutBuffer,
	}
}

func (s *Socket) ReadPacket(handlerPacket func(packet []byte)) error {
	for {
		// b, e := s.read(s.handlerGetBuffer())
		s.readBuffer.Reset()
		b, e := s.read(s.readBuffer)
		if e != nil {
			// s.handlerPutBuffer(b)
			return e
		}
		handlerPacket(b.CopyRestOfBytes())
	}
}

func (s *Socket) ReadOnePacket() ([]byte, error) {
	// b, e := s.read(s.handlerGetBuffer())
	s.readBuffer.Reset()
	b, e := s.read(s.readBuffer)
	if e != nil {
		// s.handlerPutBuffer(b)
		return nil, e
	}
	return b.CopyRestOfBytes(), nil
}

func (s *Socket) WritePacket(b []byte) error {
	if e := s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration)); e != nil {
		return e
	}

	s.writeBuffer.Reset()
	s.writeBuffer.WriteUint32(uint32(len(b)))
	s.writeBuffer.WriteBytes(b)
	_, e := s.Write(s.writeBuffer.GetBytes())
	return e
}

func (s *Socket) read(buffer *hbuffer.Buffer) (*hbuffer.Buffer, error) {
	if e := s.SetReadDeadline(time.Now().Add(s.readTimeoutDuration)); e != nil {
		return buffer, e
	}

	_, e := buffer.ReadFull(s, 4)
	if e != nil {
		return buffer, e
	}
	l := buffer.ReadUint32()
	if l > s.maxReadingBytesSize {
		return buffer, ErrOverMaxReadingSize
	}
	buffer.Reset()
	_, e = buffer.ReadFull(s, uint64(l))
	return buffer, e
}
