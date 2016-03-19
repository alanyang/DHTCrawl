package DHTCrawl

import (
	"errors"
	"log"
	"net"
)

// type Protocol struct {
// 	Addr   *net.UDPAddr
// 	Cmd    string                 //find_node, get_peers, annouce_peer, ping
// 	Params map[string]interface{} //protocol field 'a' or 'r'
// 	Type   string                 //response or query
// 	Tid    string                 //protocol field 't'
// }

type Session struct {
	Conn   *net.UDPConn
	result chan *Result
	rpc    *RPC
}

func NewSession() (*Session, error) {
	conn, err := net.ListenUDP("udp", new(net.UDPAddr))
	if err != nil {
		return nil, err
	}
	session := &Session{Conn: conn, result: make(chan *Result), rpc: NewRPC()}
	log.Printf("Start Crawl on %s", conn.LocalAddr().String())
	go session.serve()
	return session, nil
}

func (s *Session) serve() {
	for {
		buf := make([]byte, 1024)
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
