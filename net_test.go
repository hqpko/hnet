package hnet

import (
	"sync"
	"testing"

	"github.com/hqpko/hbuffer"
)

func TestNet(t *testing.T) {
	network := "tcp"
	addr := "127.0.0.1:8099"
	msg := "hello socket!"
	w := &sync.WaitGroup{}
	w.Add(1)
	go func() {
		checkTestErr(
			Listen(network,
				addr,
				func(s *Socket) {
					s.Write([]byte(msg))
				}),
			t, w)
	}()
	go func() {
		s, e := Connect(network, addr)
		checkTestErr(e, t, w)
		s.SetMaxPoolBufferSize(1 << 2)
		s.SetMaxReadingBytesSize(1 << 10)
		checkTestErr(
			s.ReadWithCallback(func(b *hbuffer.Buffer) {
				receiveMsg := string(b.GetRestOfBytes())
				if receiveMsg != msg {
					t.Errorf("reading error msg:%s", receiveMsg)
				}
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
