package DHTCrawl

import (
	// "github.com/prestonTao/upnp"
	//"runtime"
	//"string"
	// "fmt"
	"github.com/valyala/gorpc"
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
		RPCClient  *gorpc.Client
	}

	DHTConfig struct {
		RemoteServer  string //fetch metainfo server address
		Port          int    //DHT UDP listen port
		TokenValidity int    //token validity (minute)
	}
)

func NewDefaultConfig() *DHTConfig {
	return &DHTConfig{
		RemoteServer:  "127.0.0.1:1128",
		TokenValidity: 5,
		Port:          2412,
	}
}

func NewDHT(cfg *DHTConfig) *DHT {
	if cfg == nil {
		cfg = NewDefaultConfig()
	}
	session, err := NewSession(cfg.Port)
	if err != nil {
		log.Fatal(err)
	}
	return &DHT{
		Session:   session,
		Table:     NewTable(),
		Token:     NewToken(cfg.TokenValidity),
		RPCClient: gorpc.NewTCPClient(cfg.RemoteServer),
		Bootstraps: []string{
			"67.215.246.10:6881",
			"212.129.33.50:6881",
			"82.221.103.244:6881",
		},
	}
}

func (d *DHT) Run() {
	go d.Walk()

	// d.RPCClient.Start()
	// defer d.RPCClient.Stop()
	// dc := Dispatcher.NewServiceClient("FetchMetaInfo", d.RPCClient)
	for r := range d.Session.result {
		switch r.Cmd {
		case OP_FIND_NODE:
			for _, node := range r.Nodes {
				d.Table.Add(node)
			}
		case OP_GET_PEERS:
			ns := ConvertByteStream(d.Table.Last)
			d.Session.SendTo(PacketGetPeers(r.Hash, d.Table.Self, ns, d.Token.Value, r.Tid), r.UDPAddr)

		case OP_ANNOUNCE_PEER:
			if d.Token.IsValid(r.Token) {
				d.Session.SendTo(PacketAnnucePeer(r.Hash, d.Table.Self, r.Tid), r.UDPAddr)
				if d.Handler != nil {
					need := d.Handler(r.Hash)
					if need {
						//fetch metadata info from tcp port (bep_09, bep_10)
						// dc.CallAsync("Fetch", fmt.Sprintf("%X|%s", []byte(r.Hash), r.TCPAddr.String()))
						wire := NewWire()
						go wire.Download(r.Hash, r.TCPAddr)
					}
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
		if len(d.Table.Nodes) == 0 || d.Table.Nodes == nil {
			d.Join()
			time.Sleep(time.Millisecond * 800)
		} else {
			// d.Table.Mutex.Lock()
			// nodes := d.Table.Nodes[:]
			// d.Table.Mutex.Unlock()
			// for _, node := range nodes {
			// 	if node != nil {
			// 		d.Session.SendTo(PacketFindNode(node.ID.Neighbor(), NewNodeID()), node.Addr)
			// 		time.Sleep(time.Millisecond * 2)
			// 	}
			// }
			d.Table.Each(func(node *Node, _ int) {
				d.Session.SendTo(PacketFindNode(node.ID.Neighbor(), NewNodeID()), node.Addr)
				time.Sleep(time.Millisecond * 2)
			})
			d.Table.Flush()
		}
	}
}

func (d *DHT) Handle(h HandlerEnsureHash) {
	d.Handler = h
}
