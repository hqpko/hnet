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
		e := ListenSocket(network, addr, func(socket *Socket) {
			_ = socket.Write([]byte(msg))
		}, NewOption())
		_ = checkTestErr(e, t, w)
	}()
	go func() {
		s, e := ConnectSocket(network, addr, NewOption())
		if checkTestErr(e, t, w) != nil {
			return
		}
		e = s.ReadWithCallback(func(buffer *hbuffer.Buffer) {
			receiveMsg := string(buffer.GetBytes())
			if receiveMsg != msg {
				t.Errorf("reading error msg:%s", receiveMsg)
			}
			w.Done()
		})
		_ = checkTestErr(e, t, w)
	}()
	w.Wait()
}

func checkTestErr(e error, t *testing.T, w *sync.WaitGroup) error {
	if e != nil {
		w.Done()
		t.Errorf(e.Error())
	}
	return e
}
