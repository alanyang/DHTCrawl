package DHTCrawl

import (
	// "github.com/prestonTao/upnp"
	//"string"
	// "fmt"
	"log"
	"net"
	"time"
)

type (
	HashHandler func(Hash) bool

	DHT struct {
		Session         *Session
		Table           *Table
		Bootstraps      []string
		Token           *Token
		HashHandler     HashHandler
		MetadataHandler ResultHandler
		JobPool         *WireJob
		Handler         Collector
	}

	DHTConfig struct {
		RemoteServer  string //fetch metainfo server address
		Port          int    //DHT UDP listen port
		TokenValidity int    //token validity (minute)
		JobSize       int
		Entries       []string
	}

	Collector interface {
		MetaInfoHandler(*MetadataResult)
		HashHandler(Hash)
	}

	Remote string

	RemoteServer interface {
		Format(...string) string
	}
)

var (
	// DBValueUnDownload = []byte{0x01}
	// DBValueDownloading = []byte{0x02}
	DBValueUnIndexID = []byte{0x00, 0x00}
)

func NewDefaultConfig() *DHTConfig {
	return &DHTConfig{
		TokenValidity: 5,
		Port:          2412,
		JobSize:       500,
		Entries: []string{
			"67.215.246.10:6881",
			"212.129.33.50:6881",
			"82.221.103.244:6881",
		},
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
		Session:    session,
		Table:      NewTable(),
		Token:      NewToken(cfg.TokenValidity),
		JobPool:    NewWireJob(cfg.JobSize),
		Bootstraps: cfg.Entries,
	}
}

func (d *DHT) Run() {
	go d.Walk()
	go func() {
		for data := range d.JobPool.Result {
			if d.MetadataHandler != nil {
				d.MetadataHandler(data)
			}
		}
	}()

	// d.RPCClient.Start()
	// defer d.RPCClient.Stop()
	// dc := Dispatcher.NewServiceClient("FetchMetaInfo", d.RPCClient)
	for {
		r := <-d.Session.result
		switch r.Cmd {
		case OP_FIND_NODE:
			for _, node := range r.Nodes {
				if node.ID.Hex() != d.Table.Self.Hex() && d.Session.ExternalIP != node.Addr.IP.String() {
					d.Table.Add(node)
				}
			}

		case OP_PING:
			d.Session.SendTo(PacketPong(r.ID, d.Table.Self, r.Tid), r.UDPAddr)

		case OP_GET_PEERS:
			ns := ConvertByteStream(d.Table.Last)
			d.Session.SendTo(PacketGetPeers(r.Hash, r.ID, d.Table.Self, ns, d.Token.Value, r.Tid), r.UDPAddr)

		case OP_ANNOUNCE_PEER:
			if d.Token.IsValid(r.Token) {
				d.Session.SendTo(PacketAnnucePeer(r.Hash, r.ID, d.Table.Self, r.Tid), r.UDPAddr)
				if d.HashHandler != nil {
					need := d.HashHandler(r.Hash)
					if need {
						//fetch metadata info from tcp port (bep_09, bep_10)
						d.JobPool.Add(NewJob(r.Hash, r.TCPAddr))
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
			time.AfterFunc(time.Millisecond*800, d.Walk)
		} else {
			d.Table.Each(func(node *Node, _ int) {
				d.Session.SendTo(PacketFindNode(node.ID.Neighbor(d.Table.Self), NewNodeID()), node.Addr)
			})
			d.Table.Flush()
		}
	}
}

func (d *DHT) HandleHash(h HashHandler) {
	d.HashHandler = h
}

func (d *DHT) HandleMetadata(h ResultHandler) {
	d.MetadataHandler = h
}
