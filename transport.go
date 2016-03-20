package DHTCrawl

import (
	"log"
	"net"
)

type Transport struct {
	Conn *net.TCPConn
	Hash Hash
}

func NewTransport(addr *net.TCPAddr, h Hash) (*Transport, error) {
	tp := new(Transport)
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	tp.Conn = conn
	tp.Hash = h
	go tp.ReadData()
	return tp, nil
}

func (t *Transport) ReadData() {
	buf := make([]byte, 2048)
	for {
		n, err := t.Conn.Read(buf[:])
		if err != nil {
			break
		}
		log.Println(buf[:n])
	}
}
