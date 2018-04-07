package hnet

import "net"

func Listen(network, addr string, callback func(socket *Socket)) error {
	listener, err := net.Listen(network, addr)
	if err != nil {
		return err
	}
	for {
		c, e := listener.Accept()
		if e != nil {
			return e
		}
		callback(createSocket(c))
	}
}

func Connect(network, addr string) (*Socket, error) {
	c, e := net.Dial(network, addr)
	if e != nil {
		return nil, e
	}
	return createSocket(c), nil
}
