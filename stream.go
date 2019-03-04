package hnet

import (
	"github.com/hqpko/hconcurrent"
)

type Stream struct {
	socket            *Socket
	mainChannel       *hconcurrent.Concurrent
	handlerWriteError func(err error)
}

func NewStream(socket *Socket, channelSize int, handlerWriteError func(err error)) *Stream {
	stream := &Stream{socket: socket, handlerWriteError: handlerWriteError}
	stream.mainChannel = hconcurrent.NewConcurrent(channelSize, 1, stream.handlerWrite)
	stream.mainChannel.Start()
	return stream
}

func (s *Stream) Input(b []byte) bool {
	return s.mainChannel.Input(b)
}

func (s *Stream) MustInput(b []byte) {
	s.mainChannel.MustInput(b)
}

func (s *Stream) handlerWrite(i interface{}) interface{} {
	if b, ok := i.([]byte); ok {
		if err := s.socket.Write(b); err != nil {
			s.handlerWriteError(err)
		}
	}
	return nil
}

func (s *Stream) Stop() error {
	s.mainChannel.Stop()
	return s.socket.Close()
}
