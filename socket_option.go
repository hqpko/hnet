package hnet

import (
	"time"

	"github.com/hqpko/hbuffer"
)

const (
	defMaxReadingByteSize = 1 << 12 // 4k
	defTimeoutDuration    = 8 * time.Second
)

var (
	defHandlerGetBuffer = func() *hbuffer.Buffer { return hbuffer.NewBuffer() }
	defHandlerPutBuffer = func(buffer *hbuffer.Buffer) {}
)

type Option struct {
	maxReadingByteSize   uint32
	readTimeoutDuration  time.Duration
	writeTimeoutDuration time.Duration
	handlerGetBuffer     func() *hbuffer.Buffer
	handlerPutBuffer     func(buffer *hbuffer.Buffer)
}

func NewOption() *Option {
	return &Option{
		maxReadingByteSize:   defMaxReadingByteSize,
		readTimeoutDuration:  defTimeoutDuration,
		writeTimeoutDuration: defTimeoutDuration,
		handlerGetBuffer:     defHandlerGetBuffer,
		handlerPutBuffer:     defHandlerPutBuffer,
	}
}

func (o *Option) MaxReadingByteSize(maxSize uint32) *Option {
	o.maxReadingByteSize = maxSize
	return o
}

func (o *Option) ReadTimeoutDuration(timeoutDuration time.Duration) *Option {
	o.readTimeoutDuration = timeoutDuration
	return o
}

func (o *Option) WriteTimeoutDuration(timeoutDuration time.Duration) *Option {
	o.writeTimeoutDuration = timeoutDuration
	return o
}

func (o *Option) HandlerGetBuffer(handler func() *hbuffer.Buffer) *Option {
	o.handlerGetBuffer = handler
	return o
}

func (o *Option) HandlerPutBuffer(handler func(*hbuffer.Buffer)) *Option {
	o.handlerPutBuffer = handler
	return o
}
