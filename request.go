package DHTCrawl

import (
	"bytes"
)

const (
	MESSAGE_PROTOCOL = "BitTorrent protocol"
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
