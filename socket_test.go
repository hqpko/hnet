package hnet

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/hqpko/hbuffer"
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

func TestSocket(t *testing.T) {
	client, server := testConn()
	n := 10
	testData := make([][]byte, n)
	for i := 0; i < n; i++ {
		testData[i] = []byte{byte(i)}
	}
	go func() {
		for i := 0; i < n; i++ {
			if err := client.WritePacket(testData[i]); err != nil {
				t.Errorf("write packet error:%s", err.Error())
			}
		}
	}()
	for i := 0; i < n; i++ {
		if resp, err := server.ReadOnePacket(); err != nil || !bytes.Equal(testData[i], resp) {
			t.Errorf("read packet %d error:%v resp:%v shouldBe:%v", i, err, resp, testData[i])
		}
	}

}

func TestSocketReadIncompleteBytes(t *testing.T) {
	client, server := testConn()
	testData := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	go func() {
		buf := hbuffer.NewBuffer()
		buf.WriteEndianUint32(uint32(len(testData)))
		buf.WriteBytes(testData[:len(testData)/2])
		_ = client.WriteBuffer(buf)
		time.Sleep(time.Second)

		buf.Reset()
		buf.WriteBytes(testData[len(testData)/2:])
		buf.WriteEndianUint32(uint32(len(testData)))
		buf.WriteBytes(testData[:len(testData)/2])
		time.Sleep(time.Second)

		buf.WriteBytes(testData[len(testData)/2:])
		buf.WriteEndianUint32(uint32(len(testData)))
		buf.WriteBytes(testData[:len(testData)/2])
		_ = client.WriteBuffer(buf)

		buf.Reset()
		time.Sleep(time.Second)
		buf.WriteBytes(testData[len(testData)/2:])
		_ = client.WriteBuffer(buf)
	}()

	if resp, _ := server.ReadOnePacket(); !bytes.Equal(testData, resp) {
		t.Errorf("read incomplete bytes fail.")
	}

	if resp, _ := server.ReadOnePacket(); !bytes.Equal(testData, resp) {
		t.Errorf("read incomplete bytes fail.")
	}
	if resp, _ := server.ReadOnePacket(); !bytes.Equal(testData, resp) {
		t.Errorf("read incomplete bytes fail.")
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
