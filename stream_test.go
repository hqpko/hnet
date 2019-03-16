package hnet

import (
	"testing"
	"time"
)

func TestStream(t *testing.T) {
	network := "tcp"
	addr := testGetAddr()
	msg := "hello socket!"

	go func() {
		_ = ListenSocket(network, addr, func(socket *Socket) {
			stream := NewStream(socket, 16, func(err error) {
				t.Error(err)
			})
			stream.MustInput([]byte(msg))
		}, NewOption())
	}()

	time.Sleep(100 * time.Millisecond)
	s, e := ConnectSocket(network, addr, NewOption())
	if e != nil {
		t.Fatal(e)
	}
	_ = s.ReadPacket(func(packet []byte) {
		receiveMsg := string(packet)
		if receiveMsg != msg {
			t.Errorf("reading error msg:%s", receiveMsg)
		}
		_ = s.Close()
	})
}
