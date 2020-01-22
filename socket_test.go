package hnet

import (
	"net"
	"testing"
)

func BenchmarkSocket_Read_Write_1K(b *testing.B) {
	benchmarkSocketReadWrite(b, 1024)
}

func BenchmarkSocket_Read2_Write_1K(b *testing.B) {
	benchmarkSocketRead2Write(b, 1024)
}

func BenchmarkSocket_Read_Write_4K(b *testing.B) {
	benchmarkSocketReadWrite(b, 4*1024)
}

func BenchmarkSocket_Read2_Write_4K(b *testing.B) {
	benchmarkSocketRead2Write(b, 4*1024)
}

func BenchmarkSocket_Read_Write_16K(b *testing.B) {
	benchmarkSocketReadWrite(b, 16*1024)
}

func BenchmarkSocket_Read2_Write_16K(b *testing.B) {
	benchmarkSocketRead2Write(b, 16*1024)
}

func BenchmarkSocket_Read_Write_64K(b *testing.B) {
	benchmarkSocketReadWrite(b, 64*1024)
}

func BenchmarkSocket_Read2_Write_64K(b *testing.B) {
	benchmarkSocketRead2Write(b, 64*1024)
}

func benchmarkSocketReadWrite(b *testing.B, size int) {
	testData := make([]byte, size)
	client, server := testConn()
	n := b.N
	go func() {
		for i := 0; i < n; i++ {
			client.WritePacket(testData)
		}
	}()
	b.ResetTimer()
	for i := 0; i < n; i++ {
		server.read()
	}
}

func benchmarkSocketRead2Write(b *testing.B, size int) {
	testData := make([]byte, size)
	client, server := testConn()
	n := b.N
	go func() {
		for i := 0; i < n; i++ {
			client.WritePacket(testData)
		}
	}()
	b.ResetTimer()
	for i := 0; i < n; i++ {
		server.read2()
	}
}

func testConn() (client *Socket, server *Socket) {
	c := make(chan *Socket)
	if listen, err := net.Listen("tcp", "localhost:0"); err != nil {
		panic(err)
	} else {
		addr := listen.Addr().String()
		go func() {
			if conn, err := listen.Accept(); err != nil {
				panic(err)
			} else {
				c <- NewSocket(conn)
			}
		}()

		client, err = ConnectSocket("tcp", addr)
		server = <-c
		return
	}
}
