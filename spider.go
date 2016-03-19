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
	HandlerUnsureHash func(Hash, *net.UDPAddr)
	HandlerEnsureHash func(Hash, *net.UDPAddr, *net.TCPAddr)
	HandlerFoundNodes func([]*Node, *net.UDPAddr)

	DHT struct {
		Session    *Session
		Table      *Table
		Bootstraps []string
		Token      *Token
		UnsureHash HandlerUnsureHash
		EnsureHash HandlerEnsureHash
		FoundNodes HandlerFoundNodes
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
			if d.FoundNodes != nil {
				d.FoundNodes(r.Nodes, r.UDPAddr)
			}
		case OP_GET_PEERS:
			ns := ConvertByteStream(d.Table.Last)
			d.Session.SendTo(PacketGetPeers(r.Hash, d.Table.Self, ns, d.Token.Value, r.Tid), r.UDPAddr)
			if d.UnsureHash != nil {
				d.UnsureHash(r.Hash, r.UDPAddr)
			}
		case OP_ANNOUNCE_PEER:
			d.Session.SendTo(PacketAnnucePeer(r.Hash, d.Table.Self, r.Tid), r.UDPAddr)
			if d.EnsureHash != nil {
				d.EnsureHash(r.Hash, r.UDPAddr, r.TCPAddr)
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
			}
			d.Table.Nodes = nil
			time.Sleep(time.Second * 1)
		}
	}
}
