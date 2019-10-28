package hnet

import (
	"net"
	"time"
)

func Listen(network, addr string, callback func(conn net.Conn)) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	var tempDelay time.Duration
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		callback(conn)
	}
}

func Connect(network, addr string) (net.Conn, error) {
	return net.Dial(network, addr)
}

func ListenSocket(network, addr string, callback func(socket *Socket)) error {
	return Listen(network, addr, func(conn net.Conn) {
		callback(NewSocket(conn))
	})
}

func ConnectSocket(network, addr string) (*Socket, error) {
	c, e := net.Dial(network, addr)
	if e != nil {
		return nil, e
	}
	return NewSocket(c), nil
}

// KeepConnectSocket，保持对地址 addr 的连接
// reconnectDuration 断线重连的时间间隔
// fConnected 连接成功的回调
// fConnectError 连接错误的回调
// fRetry 是否继续保持连接的判断函数
func KeepConnectSocket(network, addr string, reconnectDuration time.Duration,
	fConnected func(socket *Socket),
	fConnectError func(err error),
	fRetry func() bool) {
	for {
		if socket, err := ConnectSocket(network, addr); err == nil {
			fConnected(socket)
		} else {
			fConnectError(err)
		}
		if fRetry() {
			time.Sleep(reconnectDuration)
		} else {
			break
		}
	}
}
