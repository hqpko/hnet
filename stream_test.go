package hnet

import (
	"testing"
	"time"

	"github.com/hqpko/hbuffer"
)

var testStreamAddr = "127.0.0.1:12003"

func TestStream(t *testing.T) {
	startStreamServer()

	socket, _ := ConnectSocket("tcp", testStreamAddr)
	stream := NewStream(socket)
	defer stream.Close()

	go stream.Read()
	call := stream.NewRequest(false)
	call.Buffer().WriteBool(true)
	stream.Request(call)
	if err := call.Done(); err != nil {
		t.Fatalf("hnet: stream call done error:%s", err.Error())
	}
	if ok, err := call.Buffer().ReadBool(); !ok || err != nil {
		t.Fatalf("hnet: stream response call fail.")
	}
}

func Benchmark_Stream(b *testing.B) {
	startStreamServer()

	socket, _ := ConnectSocket("tcp", testStreamAddr)
	stream := NewStream(socket)
	defer stream.Close()

	go stream.Read()
	b.StartTimer()
	defer b.StopTimer()
	for i := 0; i < b.N; i++ {
		call := stream.NewRequest(false)
		call.Buffer().WriteBool(true)
		stream.Request(call)
		call.Done()
		if ok, err := call.Buffer().ReadBool(); !ok || err != nil {
			b.Fatalf("hnet: stream response call fail.")
		}
	}
}

func startStreamServer() {
	go func() {
		_ = ListenSocket("tcp", testStreamAddr, func(socket *Socket) {
			s := NewStream(socket)
			s.SetReadCallHandler(func(req, resp *hbuffer.Buffer) {
				resp.WriteBool(true)
				s.Response(req, resp)
			})
			go func() {
				_ = s.Read()
			}()
		})
	}()
	time.Sleep(100 * time.Millisecond)
}
