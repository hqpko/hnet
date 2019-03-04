package hnet

import (
	"sync"
	"testing"

	"github.com/hqpko/hbuffer"
	"github.com/hqpko/hpool"
)

func TestNet(t *testing.T) {
	network := "tcp"
	addr := "127.0.0.1:10033"
	msg := "hello socket!"
	w := &sync.WaitGroup{}
	w.Add(1)

	maxBufferSizeInPool := 1 << 10
	pool := hpool.NewBufferPool(1<<4, maxBufferSizeInPool)
	handlerGetBuffer := func() *hbuffer.Buffer { return pool.Get() }
	handlerPutBuffer := func(buffer *hbuffer.Buffer) { pool.Put(buffer) }
	option := NewOption().HandlerGetBuffer(handlerGetBuffer).HandlerPutBuffer(handlerPutBuffer)
	go func() {
		checkTestErr(
			ListenSocketWithOption(
				network, addr,
				func(s *Socket) { s.Write([]byte(msg)) },
				option),
			t, w)
	}()
	go func() {
		s, e := ConnectSocketWithOption(network, addr, option)
		checkTestErr(e, t, w)
		checkTestErr(
			s.ReadWithCallback(func(b *hbuffer.Buffer) {
				receiveMsg := string(b.GetRestOfBytes())
				if receiveMsg != msg {
					t.Errorf("reading error msg:%s", receiveMsg)
				}
				pool.Put(b)
				w.Done()
			}),
			t, w)
	}()
	w.Wait()
}

func checkTestErr(e error, t *testing.T, w *sync.WaitGroup) {
	if e != nil {
		w.Done()
		t.Errorf(e.Error())
	}
}
