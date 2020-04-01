package hnet

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/hqpko/hbuffer"
)

func TestSocket_WritePacket_Concurrent(t *testing.T) {
	testPacket := []byte{1, 2, 3, 4, 5}
	client, server := testConn()
	goCount := 100
	writeCount := 10000
	for i := 0; i < goCount; i++ {
		go func() {
			for i := 0; i < writeCount; i++ {
				_ = client.WritePacket(testPacket)
			}
		}()
	}
	for i := 0; i < goCount*writeCount; i++ {
		if resp, err := server.ReadOnePacket(); err != nil {
			t.Error(err)
		} else if !bytes.Equal(testPacket, resp) {
			t.Errorf("read packet error, should be %v, but %v", testPacket, resp)
		}
	}
}

func TestSocket_WriteBuffer_Concurrent(t *testing.T) {
	testPacket := []byte{1, 2, 3, 4, 5}
	client, server := testConn()
	goCount := 100
	writeCount := 10000
	for i := 0; i < goCount; i++ {
		go func() {
			for i := 0; i < writeCount; i++ {
				_ = client.WriteBuffer(hbuffer.NewBufferWithHead().WriteBytes(testPacket).UpdateHead())
			}
		}()
	}
	for i := 0; i < goCount*writeCount; i++ {
		if resp, err := server.ReadOnePacket(); err != nil {
			t.Error(err)
		} else if !bytes.Equal(testPacket, resp) {
			t.Errorf("read packet error, should be %v, but %v", testPacket, resp)
		}
	}
}

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

func BenchmarkSocket_Read_Write2_1K(b *testing.B) {
	benchmarkSocketReadWrite2(b, 1024)
}

func BenchmarkSocket_Read_Write2_4K(b *testing.B) {
	benchmarkSocketReadWrite2(b, 4*1024)
}

func BenchmarkSocket_Read_Write2_16K(b *testing.B) {
	benchmarkSocketReadWrite2(b, 16*1024)
}

func BenchmarkSocket_Read_Write2_64K(b *testing.B) {
	benchmarkSocketReadWrite2(b, 64*1024)
}

func BenchmarkSocket_Read_Write_More_64K(b *testing.B) {
	benchmarkSocketReadWriteMore(b, 64*1024)
}

func BenchmarkSocket_Read_Write2_More_64K(b *testing.B) {
	benchmarkSocketReadWriteMore2(b, 64*1024)
}

func benchmarkSocketReadWrite2(b *testing.B, size int) {
	testData := make([]byte, size)
	client, server := testConn()
	n := b.N
	go func() {
		for i := 0; i < n; i++ {
			client.writePacket2(testData)
		}
	}()
	b.ResetTimer()
	for i := 0; i < n; i++ {
		server.read()
	}
}

func benchmarkSocketReadWriteMore(b *testing.B, size int) {
	data := make([][]byte, 10)
	for i := range data {
		data[i] = make([]byte, size)
	}
	client, server := testConn()
	n := b.N
	go func() {
		for i := 0; i < n; i++ {
			client.writePackets(data...)
		}
	}()
	b.ResetTimer()
	for i := 0; i < n; i++ {
		server.read()
	}
}

func benchmarkSocketReadWriteMore2(b *testing.B, size int) {
	data := make([][]byte, 10)
	for i := range data {
		data[i] = make([]byte, size)
	}
	client, server := testConn()
	n := b.N
	go func() {
		for i := 0; i < n; i++ {
			client.writePackets2(data...)
		}
	}()
	b.ResetTimer()
	for i := 0; i < n; i++ {
		server.read()
	}
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

		client, err = ConnectSocket(addr)
		server = <-c
		return
	}
}
