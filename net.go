package hnet

import (
	"net"
	"time"

	"github.com/xtaci/kcp-go"
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

func ListenSocket(network, addr string, callback func(socket *Socket), option Option) error {
	return Listen(network, addr, func(conn net.Conn) {
		callback(NewSocket(conn, option))
	})
}

func ConnectSocket(network, addr string, option Option) (*Socket, error) {
	c, e := net.Dial(network, addr)
	if e != nil {
		return nil, e
	}
	return NewSocket(c, option), nil
}

func ListenKcpSocket(addr string, callback func(socket *Socket), funcInitKcp func(session *kcp.UDPSession), option Option) error {
	listener, err := kcp.ListenWithOptions(addr, nil, 0, 0)
	if err != nil {
		return err
	}
	for {
		kcpConn, err := listener.AcceptKCP()
		if err != nil {
			return err
		}
		funcInitKcp(kcpConn)
		callback(NewSocket(kcpConn, option))
	}
}

func ConnectKcpSocket(addr string, funcInitKcp func(session *kcp.UDPSession), option Option) (*Socket, error) {
	kcpConn, err := kcp.DialWithOptions(addr, nil, 0, 0)
	if err != nil {
		return nil, err
	}
	funcInitKcp(kcpConn)
	return NewSocket(kcpConn, option), nil
}
