package DHTCrawl

import (
	"errors"
	"log"
	"net"
)

type Session struct {
	Conn       *net.UDPConn
	result     chan *Result
	rpc        *RPC
	ExternalIP string
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
	session.ExternalIP, _ = session.GetExternalIP()
	log.Printf("Start Crawl on %s", conn.LocalAddr().String())
	go session.serve()
	return session, nil
}

func (s *Session) GetExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
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
