package DHTCrawl

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/zeebo/bencode"
	"log"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	BtProtocol   = "BitTorrent protocol"
	BtExtendedID = byte(0)
	BtMessageID  = byte(20)

	PieceSize       = 1 << 14
	MaxMetadataSize = (1 << 20) * 15

	WireConnectTimeout = 2
	WireTimeout        = 5

	MetaTypeVideo    = 1
	MetaTypeAudio    = 2
	MetaTypePicture  = 3
	MetaTypeDocument = 4
	MetaTypeZip      = 5
	MetaTypeExe      = 6
	MetaTypeOther    = 7

	EventError = iota - 1
	EventHandshake
	EventExtended
	EventPiece
	EventDone
)

var (
	//[5] = 1 as extension, [7] = 1 as dht
	BtReserved = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x01}

	VideoTypeExtensions    = []string{"avi", "rmvb", "rm", "asf", "divx", "mpg", "mpeg", "mpe", "wmv", "mp4", "mkv", "vob", "fla", "3gp"}
	AudioTypeExtensions    = []string{"mp3", "m4a", "wma"}
	PictureTypeExtensions  = []string{"jpg", "jpeg", "png", "gif", "bmp", "psd", "tiff", "tga", "eps"}
	DocumentTypeExtensions = []string{"doc", "docx", "pdf", "chm"}
	ZipTypeExtensions      = []string{"zip", "rar", "7z", "cab", "iso", "gz", "bz2"}
	ExeTypeExtensions      = []string{"exe"}
)

type (
	DataHandler   func([]byte)
	ResultHandler func(*MetadataResult)

	Files []*File

	File struct {
		Path   []string `bencode:"path" json:"path,omitempty"`
		UPath  []string `bencode:"path.utf-8" json:"upath,omitempty"`
		Length int64    `bencode:"length" json:"length,omitempty"`
		Md5sum string   `bencode:"md5sum" json:"-"`
	}

	MetadataResult struct {
		Hash          Hash        `json:"hash"`
		Hex           string      `json:"hex"`
		Length        int64       `bencode:"length" json:"length,omitempty"`
		Name          string      `bencode:"name" json:"name"`
		UName         string      `bencode:"name.utf-8" json:"uname,omitempty"`
		PieceLength   int64       `bencode:"piece length" json:"-"`
		Pieces        interface{} `bencode:"pieces" json:"-"`
		Publisher     string      `bencode:"publisher" json:"publisher,omitempty"`
		UPublisher    string      `bencode:"publisher.utf-8" json:"uublisher,omitempty"`
		PublisherUrl  string      `bencode:"publisher-url" json:"publisherUrl,omitempty"`
		UPublisherUrl string      `bencode:"publisher-url.utf-8" json:"publisherUrl,omitempty"`
		Files         []*File     `bencode:"files" json:"files,omitempty"`

		Type     int      `json:"datatype,omitempty"`
		Create   string   `json:"create,omitempty"`
		Download []string `json:"download,omitempty"`
	}

	Event struct {
		Type   int
		Hash   Hash
		Reason string
		Result *MetadataResult
	}

	Processor struct {
		Hash Hash

		Data [][]byte
		Size int

		Handler     DataHandler
		HandlerSize int

		utmetadata   int
		metadata     [][]byte
		metadataSize int
		pieceLength  int

		event chan *Event

		Conn net.Conn
	}

	Wire struct {
		Processor *Processor
		Result    chan *MetadataResult
		Idle      bool
		Job       chan *Job
		mu        *sync.RWMutex
	}
)

func GetMetaType(ext string) int {
	switch {
	case InArray(VideoTypeExtensions, ext):
		return MetaTypeVideo
	case InArray(AudioTypeExtensions, ext):
		return MetaTypeAudio
	case InArray(PictureTypeExtensions, ext):
		return MetaTypePicture
	case InArray(DocumentTypeExtensions, ext):
		return MetaTypeDocument
	case InArray(ZipTypeExtensions, ext):
		return MetaTypeZip
	case InArray(ExeTypeExtensions, ext):
		return MetaTypeExe
	}
	return MetaTypeOther
}

func GetExtension(path string) (ext string) {
	es := strings.Split(path, ".")
	return es[len(es)-1]
}

func InArray(arr []string, i string) (b bool) {
	for _, v := range arr {
		if i == v {
			b = true
		}
	}
	return
}

func NewErrorResult(hash Hash) *MetadataResult {
	return &MetadataResult{Hash: hash}
}

func NewErrorEvent(reason string, hash Hash) *Event {
	return &Event{Type: EventError, Reason: reason, Hash: hash}
}

func NewWire(c chan *MetadataResult) *Wire {
	wire := new(Wire)
	wire.Result = c
	wire.Job = make(chan *Job)
	wire.mu = new(sync.RWMutex)
	wire.Processor = &Processor{
		Data:  [][]byte{},
		event: make(chan *Event),
	}
	wire.Release()
	go wire.wait()
	return wire
}

func (f Files) Len() int {
	return len(f)
}

func (f Files) Less(i, j int) bool {
	return f[i].Length < f[j].Length
}

func (f Files) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (m *MetadataResult) String() string {
	s := []string{
		"********************************",
		m.Hash.Hex(),
		m.Hash.Magnet(),
		m.Name,
	}
	if m.Length != 0 {
		s = append(s, fmt.Sprintf("%d", m.Length))
	}
	if len(m.Files) != 0 {
		s = append(s, "========FILES==========")
		for _, f := range m.Files {
			s = append(s, fmt.Sprintf("\t%s (%d)", strings.Join(f.Path, "/"), f.Length))
		}
		s = append(s, "=======================")
	}
	s = append(s, "********************************\n\n")
	return strings.Join(s, "\n")
}

func (w *Wire) Release() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Idle = true
}

func (w *Wire) Acquire() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Idle = false
}

func (w *Wire) IsIdle() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Idle
}

func (w *Wire) wait() {
	for {
		job := <-w.Job
		w.Acquire()
		w.Download(job.Hash, job.Addr)
	}
}

func (w *Wire) Download(hash Hash, addr *net.TCPAddr) (result *MetadataResult, err error) {
	defer w.Release()
	result, err = w.fromPeer(hash, addr)
	if err == nil {
		w.Result <- result
		return
	}

	result, err = w.fromHTTP(hash)
	if err == nil {
		w.Result <- result
		return
	}
	return
}

func (w *Wire) fromPeer(hash Hash, addr *net.TCPAddr) (*MetadataResult, error) {
	conn, err := net.DialTimeout("tcp", addr.String(), time.Second*WireConnectTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Second * WireTimeout))
	w.Processor.Conn = conn
	w.Processor.Start(hash)
	go func(conn net.Conn) {
		var (
			n   int
			err error
		)
		defer func() {
			if r := recover(); r != nil {
				log.Fatal(fmt.Printf("Has panic %s, received n=%d", r, n))
			}
		}()
		for {
			buf := make([]byte, 1024)
			n, err = conn.Read(buf)
			if err != nil {
				return
			}
			if n < 0 || n > 1024 {
				return
			}
			w.Processor.Write(buf[:n])
		}
	}(conn)
	timeout := time.After(time.Second * (WireTimeout + 1))
	for {
		select {
		case event := <-w.Processor.event:
			switch event.Type {
			case EventError:
				return nil, errors.New(event.Reason)
			case EventDone:
				return event.Result, nil
			case EventHandshake:
			case EventExtended:
			case EventPiece:
			}
		case <-timeout:
			return nil, errors.New("TCP timeout")
		}
	}
	return nil, errors.New("Socket timeout")
}

func (w *Wire) fromHTTP(hash Hash) (*MetadataResult, error) {
	hex := hash.Hex()
	url := fmt.Sprintf("%s/%s/%s/%s.torrent", Url, hex[:2], hex[38:], hex)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", Url)
	client := http.Client{
		Timeout: time.Duration(time.Second * 2),
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dec := bencode.NewDecoder(resp.Body)
	metadata := new(MetaData)
	err = dec.Decode(metadata)
	if err != nil {
		return nil, err
	}
	metadata.MetaInfo.Hash = hash
	return metadata.MetaInfo, nil
}

func (p *Processor) Write(data []byte) (int, error) {
	p.Size += len(data)
	p.Data = append(p.Data, data)
	for p.Size >= p.HandlerSize {
		buf := bytes.Join(p.Data, []byte{})
		p.Size -= p.HandlerSize
		if p.Size == 0 {
			p.Data = [][]byte{}
		} else {
			p.Data = [][]byte{buf[p.HandlerSize:]}
		}
		p.Handler(buf[:p.HandlerSize])
	}
	return len(data), nil
}

func (p *Processor) Start(hash Hash) {
	p.Hash = hash
	p.push(p.packetHandshakeData())
	p.handleHandshake()
}

func (p *Processor) process(size int, handler DataHandler) {
	p.HandlerSize = size
	p.Handler = handler
}

func (p *Processor) End(reason string) {
	p.event <- NewErrorEvent(reason, p.Hash)
}

func (p *Processor) handleHandshake() {
	p.process(1, func(data []byte) {
		length := int(data[0])
		p.process(length+48, func(data []byte) {
			protocol := data[:length]
			if string(protocol) != BtProtocol {
				p.End("this is not BitTorrent protocol")
				return
			}
			reserved := data[length:]
			if reserved[5]&0x10 == 0 {
				p.End("peer reject")
				return
			}
			p.event <- &Event{Type: EventHandshake}
			p.process(4, p.handleHead)
			p.push(p.packetExtendedData())
		})
	})
}

func (p *Processor) handleHead(data []byte) {
	var length uint32
	binary.Read(bytes.NewReader(data), binary.BigEndian, &length)
	if int(length) > 0 {
		p.process(int(length), p.handleBody)
	}
}

func (p *Processor) handleBody(data []byte) {
	p.process(4, p.handleHead)
	if data[0] == BtMessageID {
		p.handleExtended(data[1], data[2:])
	}
}

func (p *Processor) handleExtended(ext byte, data []byte) {
	if ext == byte(0) {
		val := make(map[string]interface{})
		err := bencode.DecodeBytes(data, &val)
		if err != nil {
			p.End(fmt.Sprintf("decode extended meta info error %s", err.Error()))
			return
		}
		p.handleExtHandshake(val)
	} else {
		p.handlePiece(data)
	}
}

func (p *Processor) handleExtHandshake(ext map[string]interface{}) {
	p.event <- &Event{Type: EventExtended}
	if size, ok := ext["metadata_size"].(int64); ok {
		if m, ok := ext["m"].(map[string]interface{}); ok {
			if meta, ok := m["ut_metadata"].(int64); ok {
				p.utmetadata = int(meta)

				if p.utmetadata == 0 || size <= 0 || size > MaxMetadataSize {
					p.End(fmt.Sprintf("extended invalid metadata_size:%d, ut_metadata:%d", size, p.utmetadata))
					return
				}

				pieceLength := int(math.Ceil(float64(size) / float64(PieceSize)))
				p.metadata = make([][]byte, pieceLength)
				for i := 0; i < pieceLength; i++ {
					p.push(p.packetPieceRequestData(i))
				}
			}
		}
	}
}

func (p *Processor) handlePiece(data []byte) {
	p.event <- &Event{Type: EventPiece}
	i := bytes.Index(data, []byte{101, 101}) + 2
	if i == 1 {
		p.End("invalid piece info dict")
		return
	}
	info := make(map[string]interface{})
	err := bencode.DecodeBytes(data[0:i], &info)
	if err != nil {
		p.End(fmt.Sprintf("decode piece dict error, %s", err.Error()))
		return
	}
	piece := data[i:]

	if t, ok := info["msg_type"].(int64); !ok || t != int64(1) {
		p.End(fmt.Sprintf("invalid msg_type: %d", t))
		return
	}

	n, ok := info["piece"].(int64)
	if !ok {
		p.End("invalid piece")
		return
	}

	if len(piece) > PieceSize {
		p.End("invalid piece size")
		return
	}

	p.metadata[int(n)] = piece
	if p.isDone() {
		p.handleDone()
	}
}

func (p *Processor) isDone() (b bool) {
	for _, piece := range p.metadata {
		if len(piece) == 0 {
			return false
		}
	}
	return true
}

func (p *Processor) handleDone() {
	data := bytes.Join(p.metadata, []byte{})
	s := sha1.Sum(data)
	if p.Hash.Hex() == fmt.Sprintf("%X", s) {
		result := new(MetadataResult)
		decoder := bencode.NewDecoder(bytes.NewReader(data))
		err := decoder.Decode(&result)
		if err != nil {
			p.End(fmt.Sprintf("Decode metadata error %s", err.Error()))
			return
		}
		result.Hash = p.Hash
		p.Conn.Close()
		p.event <- &Event{Type: EventDone, Result: result}
	}
}

func (p *Processor) packetHandshakeData() []byte {
	data := bytes.NewBuffer([]byte{})
	data.WriteByte(byte(0x13))
	data.WriteString(BtProtocol)
	data.Write(BtReserved)
	data.WriteString(string(p.Hash))
	data.Write([]byte(NewNodeID()))
	return data.Bytes()
}

func (p *Processor) packetExtendedData() []byte {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(BtMessageID)
	body.WriteByte(BtExtendedID)

	meta, _ := bencode.EncodeBytes(map[string]interface{}{"m": map[string]interface{}{"ut_metadata": 1}})
	body.Write(meta)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())

	return data.Bytes()
}

func (p *Processor) packetPieceRequestData(i int) []byte {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(BtMessageID)
	body.WriteByte(byte(p.utmetadata))

	meta, _ := bencode.EncodeBytes(map[string]interface{}{"msg_type": 0, "piece": i})
	body.Write(meta)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())

	return data.Bytes()
}

func (p *Processor) push(b []byte) {
	p.Conn.Write(b)
}
