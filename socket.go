package hnet

import (
	"errors"
	"net"
	"time"

	"github.com/hqpko/hbuffer"
)

var ErrOverMaxReadingSize = errors.New("over max reading size")

const (
	defMaxReadingByteSize = 1 << 26 // 64M
	defTimeoutDuration    = 8 * time.Second
)

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

func NewSocket(conn net.Conn) *Socket {
	return &Socket{
		Conn:                 conn,
		maxReadingBytesSize:  defMaxReadingByteSize,
		readTimeoutDuration:  defTimeoutDuration,
		writeTimeoutDuration: defTimeoutDuration,
		readBuffer:           hbuffer.NewBuffer(),
		writeBuffer:          hbuffer.NewBuffer(),
	}
}

func (s *Socket) SetMaxReadingBytesSize(size int) {
	s.maxReadingBytesSize = size
}

func (s *Socket) SetTimeoutDuration(readTimeoutDuration, writeTimeoutDuration time.Duration) {
	s.readTimeoutDuration = readTimeoutDuration
	s.writeTimeoutDuration = writeTimeoutDuration
}

func (s *Socket) ReadPacket(handlerPacket func(packet []byte)) error {
	for {
		if bytes, e := s.ReadOnePacket(); e != nil {
			return e
		} else {
			handlerPacket(bytes)
		}
	}
}

func (s *Socket) ReadOnePacket() ([]byte, error) {
	if bytes, e := s.read(); e != nil {
		return nil, e
	} else {
		data := make([]byte, len(bytes))
		copy(data, bytes)
		return data, nil
	}
}

func (s *Socket) ReadBuffer(handlerBuffer func(buffer *hbuffer.Buffer), handlersGetBuffer ...func() *hbuffer.Buffer) error {
	var handlerGetBuffer func() *hbuffer.Buffer
	if len(handlersGetBuffer) > 0 {
		handlerGetBuffer = handlersGetBuffer[0]
	} else {
		handlerGetBuffer = func() *hbuffer.Buffer { return hbuffer.NewBuffer() }
	}

	for {
		buffer := handlerGetBuffer()
		if e := s.ReadOneBuffer(buffer); e != nil {
			return e
		}
		handlerBuffer(buffer)
	}
}

func (s *Socket) ReadOneBuffer(buffer *hbuffer.Buffer) error {
	if bytes, err := s.read(); err != nil {
		return err
	} else {
		buffer.WriteBytes(bytes)
		buffer.SetPosition(0)
	}
	return nil
}

func (s *Socket) WritePacket(b []byte) error {
	s.writeBuffer.Reset()
	s.writeBuffer.WriteEndianUint32(uint32(len(b)))
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
	s.writeBuffer.WriteEndianUint32(uint32(len(b)))
	if _, e := s.Write(s.writeBuffer.GetBytes()); e != nil {
		return e
	}

	s.writeBuffer.Reset()
	s.writeBuffer.WriteBytes(b)
	_, e := s.Write(s.writeBuffer.GetBytes())
	return e
}

func (s *Socket) read() ([]byte, error) {
	if e := s.SetReadDeadline(time.Now().Add(s.readTimeoutDuration)); e != nil {
		return nil, e
	}
	if _, e := s.readBuffer.ReadFull(s, 4); e != nil {
		return nil, e
	} else if l, _ := s.readBuffer.ReadEndianUint32(); int(l) > s.maxReadingBytesSize {
		return nil, ErrOverMaxReadingSize
	} else {
		s.readBuffer.Reset()
		_, e = s.readBuffer.ReadFull(s, int(l))
		return s.readBuffer.GetBytes(), e
	}
}
