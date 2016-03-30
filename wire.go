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
	StepHandshake = iota
	StepExtension
	StepPiece
	StepDone
	StepOver
)

const (
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
	Client struct {
		Addr *net.TCPAddr
		Hash Hash
	}
	Wire struct {
		Conn      net.Conn
		Hash      Hash
		chunk     []byte
		step      int
		umetadata int
		size      int
		metaChunk [][]byte
		recvChunk int
		Result    chan map[string]interface{}
	}

	Transport struct {
		ClientChan chan *Client
	}
)

func NewTransport() *Transport {
	t := &Transport{ClientChan: make(chan *Client, 500)}
	go t.forever()
	return t
}

func (t *Transport) forever() {
	for {
		cl := <-t.ClientChan
		w, err := NewWire(cl.Hash, cl.Addr)
		if err != nil {
			continue
		}
		w.SendHandshake()
	}
}

func NewWire(hash Hash, addr *net.TCPAddr) (*Wire, error) {
	conn, err := net.DialTimeout("tcp", addr.String(), time.Second*2)
	if err != nil {
		return nil, err
	}
	wire := &Wire{Conn: conn, chunk: []byte{}, Hash: hash, Result: make(chan map[string]interface{})}
	go wire.read()
	return wire, nil
}

func (w *Wire) SendHandshake() {
	data := bytes.NewBuffer([]byte{})
	data.WriteByte(byte(len(BtProtocol)))
	data.WriteString(BtProtocol)
	data.Write(BtReserved)
	data.WriteString(string(w.Hash))
	data.Write([]byte(NewNodeID()))
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
	body.WriteByte(byte(w.umetadata))

	meta, _ := bencode.EncodeBytes(map[string]interface{}{"msg_type": 0, "piece": p})
	body.Write(meta)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())

	w.Conn.Write(data.Bytes())
	w.step = StepPiece
}

func (w *Wire) read() {
	buf := make([]byte, 2048)
	for w.step != StepOver && w.step != StepDone {
		n, err := w.Conn.Read(buf)
		if err != nil {
			//fail
			// log.Println(err, "read error!!!")
			break
		}
		// log.Println(string(buf[:n]), "Recv!!")
		w.chunk = append(w.chunk, buf[:n]...)
		w.parse()
	}
	w.Conn.Close()
}

func (w *Wire) parse() {
	var err error
	switch w.step {
	case StepHandshake:
		err = w.handleHandshake()
	case StepExtension:
		err = w.handleMessage()
	case StepPiece:
		err = w.handleMessage()
	}
	if err != nil {
		log.Println(err.Error())
		w.handleOver()
		w.Conn.Close()
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
	w.chunk = []byte{}
	w.SendExtension()
	return nil
}

func (w *Wire) handleMessage() error {
	if len(w.chunk) < 4 {
		return nil
	}
	r := bytes.NewReader(w.chunk)
	var length uint32
	binary.Read(r, binary.BigEndian, &length)
	if uint32(len(w.chunk)) < length-uint32(4) {
		return nil
	}
	// log.Println(string(w.chunk[:pl-4]))
	// log.Println(w.chunk[:20])
	mid, _ := r.ReadByte()
	if mid != BtExtensionID {
		if w.step == StepPiece {
			log.Println("***********")
			log.Println(w.chunk)
			log.Println(string(w.chunk))
		}
		w.step = StepOver
		return errors.New("Unknow protocol id")
	}

	ext, _ := r.ReadByte()
	body := make([]byte, length-2)
	r.Read(body)

	if ext == byte(0) {
		meta := make(map[string]interface{})
		err := bencode.DecodeBytes(body, &meta)
		if err != nil {
			w.step = StepOver
			return errors.New("Decode meta error")
		}
		w.chunk = []byte{}
		w.handleExtension(meta)
	} else {
		log.Println("into piece handler")
		w.chunk = w.chunk[length+4:]
		w.handlePiece(body)
	}
	return nil
}

func (w *Wire) handleExtension(ext map[string]interface{}) {
	var num int
	if size, ok := ext["metadata_size"].(int64); ok {
		if m, ok := ext["m"].(map[string]interface{}); ok {
			if meta, ok := m["ut_metadata"].(int64); ok {
				w.umetadata = int(meta)
				w.size = int(size)
				num = int(math.Ceil(float64(w.size) / float64(PieceLength)))
				log.Println(w.size, PieceLength, num)
				w.metaChunk = [][]byte{}
				for i := 0; i < num; i++ {
					w.metaChunk = append(w.metaChunk, []byte{})
				}
			}
		}
	}
	if w.umetadata == 0 || w.size == 0 || w.size > MaxMetaSize {
		w.step = StepOver
		return
	}
	log.Println("recv extension done! request piece", w.Conn.RemoteAddr().String())
	for i := 0; i < num; i++ {
		w.RequestPiece(i)
	}
}

func (w *Wire) handlePiece(b []byte) {
	bs := bytes.Split(b, []byte{101, 101})
	msg := make(map[string]interface{})
	err := bencode.DecodeBytes(append(bs[0], []byte{101, 101}...), &msg)
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
	log.Println("recv piece done!", w.Conn.RemoteAddr().String(), w.recvChunk, len(w.metaChunk))
	if len(w.metaChunk) == w.recvChunk {
		w.step = StepDone
		w.handleDone()
	}
}

func (w *Wire) handleDone() {
	b := bytes.Join(w.metaChunk, []byte{})
	meta := map[string]interface{}{}
	err := bencode.DecodeBytes(b, &meta)
	if err != nil {
		log.Printf("decode metadata %s", err.Error())
		w.step = StepOver
		return
	}
	for k, v := range meta {
		log.Println("*********")
		log.Println(k)
		log.Println(v)
	}
	w.Result <- meta
}

func (w *Wire) handleOver() {
	log.Println("over")
	w.Result <- map[string]interface{}{}
}
