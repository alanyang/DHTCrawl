package DHTCrawl

import (
	"net"
)

type (
	Client struct {
		Addr *net.TCPAddr
		Hash Hash
	}
	Transport struct {
		ClientChan chan *Client
	}
)

func NewTransport() *Transport {
	t := &Transport{ClientChan: make(chan *Client, 500)}
	go t.forever()
	return t
}

func (t *Transport) forever() {
	for {
		cl := <-t.ClientChan
		if w := NewWire(cl.Hash, cl.Addr); w != nil {
			w.SendHandshake()
		}
	}
}
