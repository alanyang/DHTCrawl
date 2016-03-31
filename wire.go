package DHTCrawl

import (
	"bytes"
	"encoding/binary"
	"github.com/zeebo/bencode"
	"log"
	"math"
	"net"
	"time"
)

const (
	BtProtocol      = "BitTorrent protocol"
	PieceLength     = 16384
	MaxSize         = 16384 * 20
	HandshakeID     = byte(0)
	BtExtensionID   = byte(20)
	HandshakeLength = 68
	MaxMetaSize     = 10000000
)

var (
	//[5] = 1 as extension, [7] = 1 as dht
	BtReserved = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x01}
)

type Wire struct {
	chunk      []byte
	conn       net.Conn
	Hash       Hash
	utmetadata int
	metaChunk  [][]byte
	recvChunk  int
}

func NewWire(hash Hash, addr *net.TCPAddr) *Wire {
	wire := &Wire{
		Hash: hash,
	}
	conn, err := net.DialTimeout("tcp", addr.String(), time.Second*2)
	if err != nil {
		return nil
	}
	wire.conn = conn
	go wire.handle()
	return wire
}

func (w *Wire) handle() {
	handshaked := false
	buf := make([]byte, 2048)
	for {
		n, err := w.conn.Read(buf)
		if err != nil {
			// log.Println(err, w.conn.RemoteAddr().String())
			return
		}
		w.chunk = append(w.chunk, buf[:n]...)

		if handshaked {
			for len(w.chunk) >= 4 {
				var length uint32
				binary.Read(bytes.NewBuffer(w.chunk[:4]), binary.BigEndian, &length)
				log.Println(length)
				if len(w.chunk) >= int(length)+4 {
					w.handleExtended(w.chunk[4 : int(length)+4])
					w.chunk = w.chunk[int(length)+4:]
				} else {
					break
				}
			}
		} else {
			if len(w.chunk) >= HandshakeLength {
				w.handleHandshake(w.chunk[:68])
				w.chunk = w.chunk[68:]
				handshaked = true
			}
		}
	}
}

func (w *Wire) Close() {
	w.conn.Close()
}

func (w *Wire) handleHandshake(data []byte) {
	plength := int(data[0])
	protocol := data[1 : plength+1]
	if string(protocol) != BtProtocol {
		w.Close()
		return
	}
	reserved := data[plength+1 : plength+1+8]
	if reserved[5]&0x10 == 0 {
		w.Close()
		return
	}
	w.SendExtension()
	log.Println("handshake response", w.conn.RemoteAddr().String())
}

func (w *Wire) handleExtended(data []byte) {
	if data[0] == BtExtensionID {
		if data[1] == byte(0) {
			log.Println("extended response", w.conn.RemoteAddr().String())
			extendedMeta := make(map[string]interface{})
			err := bencode.DecodeBytes(data[2:], &extendedMeta)
			if err != nil {
				log.Println(err)
				w.Close()
				return
			}
			w.handleMetainfo(extendedMeta)
		} else {
			log.Println("piece response", w.conn.RemoteAddr().String())
			w.handlePiece(data[2:])
		}
	}
}

func (w *Wire) handleMetainfo(ext map[string]interface{}) {
	var num int
	if size, ok := ext["metadata_size"].(int64); ok {
		if m, ok := ext["m"].(map[string]interface{}); ok {
			if meta, ok := m["ut_metadata"].(int64); ok {
				w.utmetadata = int(meta)
				num = int(math.Ceil(float64(size) / float64(PieceLength)))
				w.metaChunk = [][]byte{}
				for i := 0; i < num; i++ {
					w.metaChunk = append(w.metaChunk, []byte{})
				}

				if w.utmetadata == 0 || size == 0 || size > MaxMetaSize {
					w.Close()
					return
				}
				for i := 0; i < num; i++ {
					w.RequestPiece(i)
				}
			}
		}
	}
}

func (w *Wire) handlePiece(b []byte) {
	bs := bytes.Split(b, []byte{101, 101})
	msg := make(map[string]interface{})
	err := bencode.DecodeBytes(append(bs[0], []byte{101, 101}...), &msg)
	if err != nil {
		w.Close()
		return
	}
	if t, ok := msg["msg_type"].(int64); ok && t != int64(1) {
		w.Close()
		return
	}
	piece, ok := msg["piece"].(int64)
	if !ok {
		w.Close()
		return
	}

	w.metaChunk[int(piece)] = bs[1]
	w.recvChunk++
	log.Println("recv piece done!", w.conn.RemoteAddr().String(), w.recvChunk, len(w.metaChunk))
	if len(w.metaChunk) == w.recvChunk {
		w.handleDone()
	}
}

func (w *Wire) handleDone() {
	b := bytes.Join(w.metaChunk, []byte{})
	meta := map[string]interface{}{}
	err := bencode.DecodeBytes(b, &meta)
	if err != nil {
		log.Printf("decode metadata %s", err.Error())
		return
	}
	for k, _ := range meta {
		log.Println(k)
	}
}

func (w *Wire) SendHandshake() {
	data := bytes.NewBuffer([]byte{})
	data.WriteByte(byte(len(BtProtocol)))
	data.WriteString(BtProtocol)
	data.Write(BtReserved)
	data.WriteString(string(w.Hash))
	data.Write([]byte(NewNodeID()))
	w.conn.Write(data.Bytes())
}

func (w *Wire) SendExtension() {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(BtExtensionID)
	body.WriteByte(HandshakeID)

	meta, _ := bencode.EncodeBytes(map[string]interface{}{"m": map[string]interface{}{"ut_metadata": 1}})
	body.Write(meta)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())

	w.conn.Write(data.Bytes())
}

func (w *Wire) RequestPiece(p int) {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(BtExtensionID)
	body.WriteByte(byte(w.utmetadata))

	meta, _ := bencode.EncodeBytes(map[string]interface{}{"msg_type": 0, "piece": p})
	body.Write(meta)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())

	w.conn.Write(data.Bytes())
}
