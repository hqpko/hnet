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

func (s *Socket) SetMaxReadingBytesSize(size int) *Socket {
	s.maxReadingBytesSize = size
	return s
}

func (s *Socket) SetTimeoutDuration(readTimeoutDuration, writeTimeoutDuration time.Duration) *Socket {
	s.readTimeoutDuration = readTimeoutDuration
	s.writeTimeoutDuration = writeTimeoutDuration
	return s
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
		buffer.WriteBytes(bytes).SetPosition(0)
	}
	return nil
}

func (s *Socket) WritePacket(b []byte) error {
	s.writeBuffer.Reset().WriteEndianUint32(uint32(len(b))).WriteBytes(b)
	return s.WriteBuffer(s.writeBuffer)
}

func (s *Socket) WriteBuffer(buffer *hbuffer.Buffer) error {
	if e := s.SetWriteDeadline(time.Now().Add(s.writeTimeoutDuration)); e != nil {
		return e
	}
	_, e := s.Write(buffer.GetBytes())
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
		_, e = s.readBuffer.Reset().ReadFull(s, int(l))
		return s.readBuffer.GetBytes(), e
	}
}

func (s *Socket) writePackets(bs ...[]byte) error {
	s.writeBuffer.Reset().WriteEndianUint32(0)
	for _, b := range bs {
		s.writeBuffer.WriteBytes(b)
	}
	return s.WriteBuffer(s.writeBuffer.UpdateHead())
}

// bench 测试结果，net.Buffers.WriteTo 使用 writev 方式，zero copy 但性能更低了，见 bench_read_write2 测试集
func (s *Socket) writePacket2(b []byte) error {
	buf := &net.Buffers{s.writeBuffer.Reset().WriteEndianUint32(uint32(len(b))).GetBytes(), b}
	_, e := buf.WriteTo(s)
	return e
}

// 即使每次发送更多的数据，net.Buffers.WriteTo 也没有更好的表现，结论是继续使用内存拷贝方式传输数据(socket.WritePacket)，见 bench_read_write_more 测试集
func (s *Socket) writePackets2(bs ...[]byte) error {
	l := 0
	buf := make(net.Buffers, len(bs)+1)
	for i, b := range bs {
		buf[i+1] = b
		l += len(b)
	}
	buf[0] = s.writeBuffer.Reset().WriteEndianUint32(uint32(l)).GetBytes()
	_, e := (&buf).WriteTo(s)
	return e
}

func (s *Socket) read2() ([]byte, error) {
	if e := s.SetReadDeadline(time.Now().Add(s.readTimeoutDuration)); e != nil {
		return nil, e
	}
	for {
		if s.readBuffer.Available() > 4 {
			l, _ := s.readBuffer.ReadEndianUint32()
			size := int(l)
			if size > s.maxReadingBytesSize {
				return nil, ErrOverMaxReadingSize
			}
			if s.readBuffer.Available() >= size {
				bytes, _ := s.readBuffer.ReadBytes(size)
				if s.readBuffer.Available() == 0 {
					s.readBuffer.Reset()
				}
				return bytes, nil
			} else if s.readBuffer.GetPosition() > 4 {
				s.readBuffer.DeleteBefore(s.readBuffer.GetPosition() - 4)
			}
		}
		s.readBuffer.SetPosition(s.readBuffer.Len())
		if _, e := s.readBuffer.ReadFromReader(s); e != nil {
			return nil, e
		}
		s.readBuffer.SetPosition(0)
	}
}
