package hnet

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func BenchmarkSocket_WritePacket(b *testing.B) {
	network := "tcp"
	addr := testGetAddr()
	go func() {
		_ = ListenSocket(network, addr, func(socket *Socket) {
			_ = socket.ReadPacket(func(packet []byte) {})
		}, DefaultOption)
	}()

	time.Sleep(100 * time.Millisecond)
	s, e := ConnectSocket(network, addr, DefaultOption)
	if e != nil {
		b.Fatal(e)
	}
	packet := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	for i := 0; i < b.N; i++ {
		_ = s.WritePacket(packet)
	}
}

func BenchmarkSocket_WritePacket2(b *testing.B) {
	network := "tcp"
	addr := testGetAddr()
	go func() {
		_ = ListenSocket(network, addr, func(socket *Socket) {
			_ = socket.ReadPacket(func(packet []byte) {})
		}, DefaultOption)
	}()

	time.Sleep(100 * time.Millisecond)
	s, _ := ConnectSocket(network, addr, DefaultOption)
	packet := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	for i := 0; i < b.N; i++ {
		_ = s.writePacket2(packet)
	}
}

func TestSocket(t *testing.T) {
	network := "tcp"
	addr := testGetAddr()
	msg := "hello socket!"
	w := &sync.WaitGroup{}
	w.Add(1)

	go func() {
		_ = ListenSocket(network, addr, func(socket *Socket) {
			_ = socket.WritePacket([]byte(msg))
		}, DefaultOption)
	}()

	time.Sleep(100 * time.Millisecond)
	s, e := ConnectSocket(network, addr, DefaultOption)
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

func testGetAddr() string {
	rand.Seed(time.Now().UnixNano())
	addr := fmt.Sprintf("127.0.0.1:%d", 10000+rand.Int31n(3000))
	return addr
}
