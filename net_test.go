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

	go func() {
		checkTestErr(
			ListenSocket(network,
				addr,
				func(s *Socket) {
					s.SetBufferPool(pool)
					s.Write([]byte(msg))
				}),
			t, w)
	}()
	go func() {
		s, e := ConnectSocket(network, addr)
		checkTestErr(e, t, w)
		s.SetBufferPool(pool)
		s.SetMaxReadingBytesSize(1 << 10)
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
