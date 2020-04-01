package hnet

import (
	"net"
)

func ListenSocket(addr string, handler func(socket *Socket)) error {
	if listener, err := net.Listen("tcp", addr); err != nil {
		return err
	} else {
		for {
			if conn, err := listener.Accept(); err != nil {
				return err
			} else {
				handler(NewSocket(conn))
			}
		}
	}
}

func ConnectSocket(addr string) (*Socket, error) {
	c, e := net.Dial("tcp", addr)
	if e != nil {
		return nil, e
	}
	return NewSocket(c), nil
}
