package DHTCrawl

import (
	"fmt"
	"github.com/anacrolix/torrent"
	"github.com/valyala/gorpc"
	"log"
	"strconv"
	"strings"
)

var (
	Dispatcher    *gorpc.Dispatcher
	TorrentClient *torrent.Client
)

type (
	FetchMetaInfoService struct {
		MaxActive int // max enabled gorouting
		Actived   int // actived fetch gorouting
	}
)

func (fs *FetchMetaInfoService) Fetch(hashPeer string) {
	log.Println(hashPeer)
	hp := strings.Split(hashPeer, "|")
	hash := hp[0]

	ipp := strings.Split(hp[1], ":")
	ip := StringToIPBytes(ipp[0])
	port, _ := strconv.Atoi(ipp[1])
	fs.fromWire(hash, ip, port)
}

func (fs *FetchMetaInfoService) fromWire(hash string, ip []byte, port int) {
	log.Println(hash, ip, port)

	t, err := TorrentClient.AddMagnet(fmt.Sprintf("magnet:?xt=urn:btih:%s", hash))
	if err != nil {
		return
	}

	t.AddPeers([]torrent.Peer{{IP: ip, Port: port}})
	<-t.GotInfo()
	t.DownloadAll()
	TorrentClient.WaitAll()
	log.Println(t.MetaInfo().Info.Name)
}

func (fs *FetchMetaInfoService) fromThunder(hash string) {
}

func NewTorrentClient() *torrent.Client {
	cli, err := torrent.NewClient(nil)
	if err != nil {
		log.Fatal(err)
	}
	return cli
}

func init() {
	Dispatcher = gorpc.NewDispatcher()
	service := &FetchMetaInfoService{MaxActive: 500}
	Dispatcher.AddService("FetchMetaInfo", service)
}
