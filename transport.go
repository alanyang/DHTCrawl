package DHTCrawl

import (
	"log"
	"net"
	"time"
)

type (
	Mail struct {
		Hash Hash
		Addr *net.TCPAddr
	}

	Wire struct {
		Poll  []*Transport
		Inbox chan *Mail
	}

	Transport struct {
		Conn net.Conn
	}
)

func NewWire() *Wire {
	w := &Wire{Poll: []*Transport{}, Inbox: make(chan *Mail, 5)}
	go w.forever()
	return w
}

func (w *Wire) AddJob(h Hash, a *net.TCPAddr) {
	w.Inbox <- &Mail{h, a}
}

func (w *Wire) forever() {
	for {
		m := <-w.Inbox
		t, err := NewTransport(m.Addr)
		if err != nil {
			log.Println(err)
			continue
		}
		t.Send(PacketHandshake(m.Hash))
	}
}

func NewTransport(addr *net.TCPAddr) (*Transport, error) {
	tp := new(Transport)
	conn, err := net.DialTimeout("tcp", addr.String(), time.Second*5)
	if err != nil {
		return nil, err
	}
	tp.Conn = conn
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
		log.Println(string(b[:n]))
		log.Fatal("end")
	}
}

func (t *Transport) Send(b []byte) {
	t.Conn.Write(b)
}
