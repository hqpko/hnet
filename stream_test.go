package hnet

import (
	"sync"
	"testing"
	"time"

	"github.com/hqpko/hbuffer"
)

var (
	testStreamAddr = "127.0.0.1:12003"
	testStreamOnce = sync.Once{}
)

func TestStream(t *testing.T) {
	startStreamServer()

	socket, _ := ConnectSocket("tcp", testStreamAddr)
	stream := NewStream(socket)
	defer stream.Close()

	go stream.Run()
	time.Sleep(100 * time.Millisecond)

	call := stream.NewCall(false)
	call.Buffer().WriteBool(true)
	stream.Call(call)
	if err := call.Done(); err != nil {
		t.Fatalf("hnet: stream call done error:%s", err.Error())
	}
	if ok, err := call.Buffer().ReadBool(); !ok || err != nil {
		t.Fatalf("hnet: stream response call fail.")
	}
}

func Benchmark_StreamCall(b *testing.B) {
	startStreamServer()

	socket, _ := ConnectSocket("tcp", testStreamAddr)
	stream := NewStream(socket)
	defer stream.Close()

	go stream.Run()
	time.Sleep(100 * time.Millisecond)

	b.StartTimer()
	defer b.StopTimer()
	for i := 0; i < b.N; i++ {
		call := stream.NewCall(false)
		call.Buffer().WriteBool(true)
		stream.Call(call)
		call.Done()
		if ok, err := call.Buffer().ReadBool(); !ok || err != nil {
			b.Fatalf("hnet: stream response call fail.")
		}
	}
}

func Benchmark_StreamMultiCall(b *testing.B) {
	startStreamServer()

	socket, _ := ConnectSocket("tcp", testStreamAddr)
	stream := NewStream(socket)
	defer stream.Close()

	go stream.Run()
	time.Sleep(100 * time.Millisecond)

	b.StartTimer()
	defer b.StopTimer()
	count := b.N
	w := sync.WaitGroup{}
	goCount := 16
	for i := 0; i < goCount; i++ {
		w.Add(1)
		go func() {
			for i := 0; i < count/goCount; i++ {
				call := stream.NewCall(false)
				call.Buffer().WriteBool(true)
				stream.Call(call)
				call.Done()
				if ok, err := call.Buffer().ReadBool(); !ok || err != nil {
					b.Fatalf("hnet: stream response call fail.")
				}
			}
			w.Done()
		}()
	}
	w.Wait()
}

func startStreamServer() {
	testStreamOnce.Do(func() {
		go func() {
			_ = ListenSocket("tcp", testStreamAddr, func(socket *Socket) {
				s := NewStream(socket)
				s.SetReadCallHandler(func(req, resp *hbuffer.Buffer) {
					resp.WriteBool(true)
					s.Reply(req, resp)
				})
				go func() {
					_ = s.Run()
				}()
			})
		}()
		time.Sleep(100 * time.Millisecond)
	})
}
