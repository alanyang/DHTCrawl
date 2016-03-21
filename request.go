package DHTCrawl

import (
	"bytes"
	"encoding/binary"
	"github.com/zeebo/bencode"
)

const (
	MAX_METADATA_SIZE = 3145728
	MESSAGE_PROTOCOL  = "BitTorrent protocol"

	MESSAGE_ID   byte = 20
	EXTENSION_ID byte = 0
)

var (
	//[5] |= 0x01 enable extended message ,[7] |= 0x01 is dht
	MESSAGE_RESERVED = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x01}
)

func PacketHandshake(hash Hash) []byte {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(byte(len(MESSAGE_PROTOCOL)))
	body.WriteString(MESSAGE_PROTOCOL)
	body.Write(MESSAGE_RESERVED)
	//hash
	body.Write([]byte(hash))
	//peerid
	body.Write([]byte(NewNodeID()))
	return body.Bytes()
}

func PacketExtension() []byte {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(MESSAGE_ID)
	body.WriteByte(EXTENSION_ID)

	m := map[string]interface{}{"m": map[string]interface{}{"ut_bmetadata": 1}}
	d, err := bencode.EncodeBytes(m)
	if err != nil {
		return []byte{}
	}
	body.Write(d)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())
	return data.Bytes()
}

func PacketPiece(metadata byte, n int) []byte {
	body := bytes.NewBuffer([]byte{})
	body.WriteByte(MESSAGE_ID)
	body.WriteByte(metadata)

	m := map[string]interface{}{"msg_type": 1, "piece": n}
	d, err := bencode.EncodeBytes(m)
	if err != nil {
		return []byte{}
	}
	body.Write(d)

	data := bytes.NewBuffer([]byte{})
	binary.Write(data, binary.BigEndian, uint32(body.Len()))
	data.Write(body.Bytes())
	return data.Bytes()
}
