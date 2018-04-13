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
