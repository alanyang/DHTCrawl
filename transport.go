package DHTCrawl

import (
	"log"
	"net"
	"time"
)

type Transport struct {
	Conn net.Conn
	Hash Hash
}

func NewTransport(addr *net.TCPAddr, h Hash) (*Transport, error) {
	tp := new(Transport)
	conn, err := net.DialTimeout("tcp", addr.String(), time.Millisecond*500)
	if err != nil {
		return nil, err
	}
	tp.Conn = conn
	tp.Hash = h
	go tp.ReadData()
	return tp, nil
}

func (t *Transport) ReadData() {
	b := make([]byte, 2048)
	for {
		n, err := t.Conn.Read(b)
		if err != nil {
			break
		}
		log.Println(b[:n])
	}
}
