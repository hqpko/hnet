package hnet

import (
	"sync"
	"testing"

	"github.com/hqpko/hbuffer"
)

func TestSocket(t *testing.T) {
	network := "tcp"
	addr := "127.0.0.1:10033"
	msg := "hello socket!"
	w := &sync.WaitGroup{}
	w.Add(1)

	go func() {
		checkTestErr(
			ListenSocket(
				network, addr,
				func(s *Socket) { s.Write([]byte(msg)) },
				NewOption()),
			t, w)
	}()
	go func() {
		s, e := ConnectSocket(network, addr, NewOption())
		checkTestErr(e, t, w)
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
