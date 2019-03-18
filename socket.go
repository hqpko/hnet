package hnet

import (
	"errors"
	"net"
	"time"

	"github.com/hqpko/hbuffer"
)

var ErrOverMaxReadingSize = errors.New("over max reading size")

type Socket struct {
	net.Conn
	maxReadingBytesSize  int
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
	}
}

func (s *Socket) ReadPacket(handlerPacket func(packet []byte)) error {
	for {
		s.readBuffer.Reset()
		if e := s.read(s.readBuffer); e != nil {
			return e
		}
		handlerPacket(s.readBuffer.CopyRestOfBytes())
	}
}

func (s *Socket) ReadBuffer(handlerBuffer func(buffer *hbuffer.Buffer), handlerGetBuffer func() *hbuffer.Buffer) error {
	for {
		buffer := handlerGetBuffer()
		if e := s.read(buffer); e != nil {
			return e
		}
		handlerBuffer(buffer)
	}
}

func (s *Socket) ReadOnePacket() ([]byte, error) {
	s.readBuffer.Reset()
	if e := s.read(s.readBuffer); e != nil {
		return nil, e
	}
	return s.readBuffer.CopyRestOfBytes(), nil
}

func (s *Socket) ReadOneBuffer(buffer *hbuffer.Buffer) error {
	return s.read(buffer)
}

func (s *Socket) WritePacket(b []byte) error {
	s.writeBuffer.Reset()
	s.writeBuffer.WriteUint32(uint32(len(b)))
	s.writeBuffer.WriteBytes(b)
	return s.WriteBuffer(s.writeBuffer)
}

func (s *Socket) WriteBuffer(buffer *hbuffer.Buffer) error {
	if e := s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration)); e != nil {
		return e
	}
	_, e := s.Write(buffer.GetBytes())
	return e
}

func (s *Socket) writePacket2(b []byte) error {
	if e := s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration)); e != nil {
		return e
	}

	s.writeBuffer.Reset()
	s.writeBuffer.WriteUint32(uint32(len(b)))
	if _, e := s.Write(s.writeBuffer.GetBytes()); e != nil {
		return e
	}

	s.writeBuffer.Reset()
	s.writeBuffer.WriteBytes(b)
	_, e := s.Write(s.writeBuffer.GetBytes())
	return e
}

func (s *Socket) read(buffer *hbuffer.Buffer) error {
	if e := s.SetReadDeadline(time.Now().Add(s.readTimeoutDuration)); e != nil {
		return e
	}

	if l, e := s.readPacketLen(buffer); e != nil {
		return e
	} else {
		if l > s.maxReadingBytesSize {
			return ErrOverMaxReadingSize
		}
		buffer.Reset()
		_, e = buffer.ReadFull(s, int(l))
		return e
	}
}

func (s *Socket) readPacketLen(buffer *hbuffer.Buffer) (int, error) {
	p := buffer.GetPosition()
	for {
		if _, e := buffer.ReadFull(s, 1); e != nil {
			return 0, e
		}
		if b, e := buffer.ReadByte(); e != nil {
			return 0, e
		} else if b >= 0x80 {
			buffer.Back(1)
		} else {
			buffer.SetPosition(p)
			break
		}
	}
	i, e := buffer.ReadUint32()
	return int(i), e
}
