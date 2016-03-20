package DHTCrawl

import (
	// "github.com/prestonTao/upnp"
	//"runtime"
	//"string"
	"log"
	"net"
	"time"
)

type (
	HandlerEnsureHash func(Hash) bool

	DHT struct {
		Session    *Session
		Table      *Table
		Bootstraps []string
		Token      *Token
		Handler    HandlerEnsureHash
	}
)

func NewDHT() *DHT {
	session, err := NewSession()
	if err != nil {
		log.Fatal(err)
	}
	return &DHT{
		Session: session,
		Table:   NewTable(),
		Token:   NewToken(20),
		Bootstraps: []string{
			"67.215.246.10:6881",
			"212.129.33.50:6881",
			"82.221.103.244:6881",
		},
	}
}

func (d *DHT) Run() {
	// if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
	// 	l := d.Session.Conn.LocalAddr().String()
	// 	ls := strings.Split(l, ":")
	// 	UDPPort, _ := strconv.Atoi(ls[len(ls)-1])
	// 	mapping := new(upnp.Upnp)
	// 	if err := mapping.AddPortMapping(UDPPort, UDPPort, "UDP"); err == nil {
	// 		log.Println("UPnP Plugin is running")
	// 	} else {
	// 		log.Fatal("UPnP start error")
	// 	}
	// }
	go d.Walk()
	for {
		r := <-d.Session.result
		switch r.Cmd {
		case OP_FIND_NODE:
			for _, node := range r.Nodes {
				d.Table.Add(node)
			}

		case OP_GET_PEERS:
			ns := ConvertByteStream(d.Table.Last)
			d.Session.SendTo(PacketGetPeers(r.Hash, d.Table.Self, ns, d.Token.Value, r.Tid), r.UDPAddr)

		case OP_ANNOUNCE_PEER:
			if d.Token.Value == r.Token {
				d.Session.SendTo(PacketAnnucePeer(r.Hash, d.Table.Self, r.Tid), r.UDPAddr)
			}
			if d.Handler != nil {
				need := d.Handler(r.Hash)
				log.Fatal(r.TCPAddr.String())
				if need {
					//fetch metadata info from tcp port (bep_09, bep_10)
				}
			}
		}
	}
}

func (d *DHT) Join() {
	for _, b := range d.Bootstraps {
		addr, err := net.ResolveUDPAddr("udp", b)
		if err != nil {
			continue
		}
		d.Session.SendTo(PacketFindNode(d.Table.Self, NewNodeID()), addr)
	}
}

func (d *DHT) Walk() {
	for {
		if len(d.Table.Nodes) == 0 {
			d.Join()
		} else {
			for _, node := range d.Table.Nodes {
				d.Session.SendTo(PacketFindNode(node.ID.Neighbor(), NewNodeID()), node.Addr)
				time.Sleep(time.Millisecond)
			}
			d.Table.Nodes = nil
		}
	}
}

func (d *DHT) Handle(h HandlerEnsureHash) {
	d.Handler = h
}
