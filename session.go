package DHTCrawl

import (
	"errors"
	"log"
	"net"
)

type Session struct {
	Conn   *net.UDPConn
	result chan *Result
	rpc    *RPC
}

func NewSession(port int) (*Session, error) {
	var addr *net.UDPAddr
	if port == 0 {
		addr = new(net.UDPAddr)
	} else {
		addr = &net.UDPAddr{IP: net.IP{0, 0, 0, 0}, Port: port}
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	if err = conn.SetWriteBuffer(1 << 20); err != nil {
		return nil, err
	}

	if err = conn.SetReadBuffer(1 << 20); err != nil {
		return nil, err
	}

	session := &Session{Conn: conn, result: make(chan *Result), rpc: NewRPC()}
	log.Printf("Start Crawl on %s", conn.LocalAddr().String())
	go session.serve()
	return session, nil
}

func (s *Session) serve() {
	buf := make([]byte, 1024)
	for {
		_, addr, err := s.Conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		r, err := s.rpc.parse(buf, addr)
		if err != nil {
			continue
		}
		s.result <- r
	}
}

func (s *Session) SendTo(data []byte, addr *net.UDPAddr) (int, error) {
	if len(data) == 0 {
		return 0, errors.New("Can't send empty []byte")
	}
	return s.Conn.WriteToUDP(data, addr)
}
