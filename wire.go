package DHTCrawl

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/zeebo/bencode"
	"log"
	"math"
	"net"
	"time"
)

const (
	StepHandshake = 0
	StepExtension = 1
	StepPiece     = 2
	StepDone      = 3
	StepOver      = 9

	BtProtocol    = "BitTorrent protocol"
	PieceLength   = 16384
	HandshakeID   = byte(0)
	BtExtensionID = byte(20)

	MaxMetaSize = 10000000
)

var (
	//[5] = 1 as extension, [7] = 1 as dht
	BtReserved = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x01}
)

type (
	Wire struct {
		Conn      net.Conn
		Hash      Hash
		chunk     []byte
		step      int
		meta      int
		size      int
		metaChunk [][]byte
		recvChunk int
		Result    chan map[string]interface{}
	}
)

func NewWire(hash Hash, addr *net.TCPAddr) (*Wire, error) {
	conn, err := net.DialTimeout("tcp", addr.String(), time.Second*3)
	if err != nil {
		return nil, err
	}
	wire := &Wire{Conn: conn, chunk: []byte{}, Hash: hash, Result: make(chan map[string]interface{})}
	go wire.read()
	wire.SendHandshake()
	return wire, nil
}

func (w *Wire) SendHandshake() {
	data := bytes.NewBuffer([]byte{})
	data.WriteByte(byte(len(BtProtocol)))
	data.WriteString(BtProtocol)
	data.Write(BtReserved)
	data.Write([]byte(w.Hash))
	data.Write([]byte(NewNodeID()))
	log.Println(data.Bytes())
	w.Conn.Write(data.Bytes())
	w.step = StepHandshake
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

	w.Conn.Write(data.Bytes())
	w.step = StepExtension
}

func (w *Wire) RequestPiece(p int) {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(BtExtensionID)
	body.WriteByte(byte(w.meta))

	meta, _ := bencode.EncodeBytes(map[string]interface{}{"msg_type": 0, "piece": p})
	body.Write(meta)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())

	w.Conn.Write(data.Bytes())
	w.step = StepPiece
}

func (w *Wire) read() {
	buf := make([]byte, 1024)
	for w.step != StepDone && w.step != StepOver {
		n, err := w.Conn.Read(buf)
		if err != nil {
			//fail
			log.Println(err)
			break
		}
		w.chunk = append(w.chunk, buf[:n]...)
		log.Println(w.chunk)
		w.parse()
	}
}

func (w *Wire) parse() {
	switch w.step {
	case StepHandshake:
		w.handleHandshake()
	case StepExtension:
		w.handleMessage()
	case StepDone:
		w.handleDone()
	case StepOver:
		w.handleOver()
	}
}

func (w *Wire) handleHandshake() error {
	if len(w.chunk) < 68 {
		return nil
	}
	r := bytes.NewReader(w.chunk[0:68])
	r.ReadByte()
	protocol := make([]byte, 19)
	r.Read(protocol)
	if string(protocol) != BtProtocol {
		w.step = StepOver
		return errors.New("Invalid protocol field")
	}
	reserved := make([]byte, 8)
	r.Read(reserved)
	if reserved[5]&0x10 == 0 {
		w.step = StepOver
		return errors.New("Peer choking")
	}
	w.chunk = w.chunk[68:]
	w.SendExtension()
	return nil
}

func (w *Wire) handleMessage() error {
	if len(w.chunk) < 4 {
		return nil
	}
	r := bytes.NewReader(w.chunk)
	var pl uint32
	binary.Read(r, binary.BigEndian, &pl)
	if uint32(len(w.chunk)) < pl-uint32(4) {
		return nil
	}
	mid, _ := r.ReadByte()
	if mid != BtExtensionID {
		w.step = StepOver
		return errors.New("Unknow protocol id")
	}

	ext, _ := r.ReadByte()
	b := make([]byte, pl-2)
	r.Read(b)
	if ext == 0 {
		meta := make(map[string]interface{})
		err := bencode.DecodeBytes(b, &meta)
		if err != nil {
			w.step = StepOver
			return errors.New("Decode meta error")
		}
		w.handleExtension(meta)
	} else {
		w.handlePiece(b)
	}
	w.chunk = w.chunk[pl+4:]
	return nil
}

func (w *Wire) handleExtension(ext map[string]interface{}) {
	var num int
	if size, ok := ext["metadata_size"].(int64); ok {
		if m, ok := ext["m"].(map[string]interface{}); ok {
			if meta, ok := m["ut_metadata"].(int64); ok {
				w.meta = int(meta)
				w.size = int(size)
				num = int(math.Ceil(float64(w.size) / float64(PieceLength)))
				w.metaChunk = [][]byte{}
				for i := 0; i < num; i++ {
					w.metaChunk = append(w.metaChunk, []byte{})
				}
			}
		}
	}
	if w.meta == 0 || w.size == 0 || w.size > MaxMetaSize {
		w.step = StepOver
		return
	}
	for i := 0; i < num; i++ {
		w.RequestPiece(i)
	}
}

func (w *Wire) handlePiece(b []byte) {
	bs := bytes.Split(b, []byte{101, 101})
	msg := make(map[string]interface{})
	err := bencode.DecodeBytes(bs[0], &msg)
	if err != nil {
		w.step = StepOver
		return
	}
	if t, ok := msg["msg_type"].(int64); ok && t != int64(1) {
		w.step = StepOver
		return
	}
	piece, ok := msg["piece"].(int64)
	if !ok {
		w.step = StepOver
		return
	}

	_, ok = msg["total_size"].(int64)
	if !ok {
		w.step = StepOver
		return
	}

	w.metaChunk[int(piece)] = bs[1]
	w.recvChunk++
	if len(w.metaChunk) == w.recvChunk {
		w.step = StepDone
		return
	}
}

func (w *Wire) handleDone() {
	b := bytes.Join(w.metaChunk, []byte{})
	meta := map[string]interface{}{}
	err := bencode.DecodeBytes(b, &meta)
	if err != nil {
		w.step = StepOver
		return
	}
	log.Println(meta)
	log.Fatal("debug")
	w.Result <- meta
}

func (w *Wire) handleOver() {
	log.Println("over")
	w.Result <- map[string]interface{}{}
}
