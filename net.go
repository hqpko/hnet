package hnet

import (
	"net"
)

func Listen(network, addr string, callback func(conn net.Conn)) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	for {
		if conn, err := listener.Accept(); err != nil {
			return err
		} else {
			callback(conn)
		}
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
