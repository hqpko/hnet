package hnet

import (
	"sync"
	"testing"
)

func TestStream(t *testing.T) {
	network := "tcp"
	addr := "127.0.0.1:10034"
	msg := "hello socket!"
	w := &sync.WaitGroup{}
	w.Add(1)

	go func() {
		e := ListenSocket(network, addr, func(socket *Socket) {
			stream := NewStream(socket, 16, func(err error) {
				t.Error(err)
			})
			stream.MustInput([]byte(msg))
		}, NewOption())
		_ = checkTestErr(e, t, w)
	}()
	go func() {
		s, e := ConnectSocket(network, addr, NewOption())
		if checkTestErr(e, t, w) != nil {
			return
		}
		e = s.ReadPacket(func(packet []byte) {
			receiveMsg := string(packet)
			if receiveMsg != msg {
				t.Errorf("reading error msg:%s", receiveMsg)
			}
			w.Done()
		})
		_ = checkTestErr(e, t, w)
	}()
	w.Wait()
}
