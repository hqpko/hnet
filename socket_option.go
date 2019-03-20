package hnet

import (
	"time"
)

const (
	defMaxReadingByteSize = 1 << 12 // 4k
	defTimeoutDuration    = 8 * time.Second
)

var DefaultOption = Option{
	MaxReadingByteSize:   defMaxReadingByteSize,
	ReadTimeoutDuration:  defTimeoutDuration,
	WriteTimeoutDuration: defTimeoutDuration,
}

type Option struct {
	MaxReadingByteSize   int
	ReadTimeoutDuration  time.Duration
	WriteTimeoutDuration time.Duration
}
