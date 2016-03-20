package DHTCrawl

import (
	"net"
	"testing"
)

func Test_Handshake(t *testing.T) {
	addr := &net.TCPAddr{IP: net.IP{49, 64, 213, 195}, Port: 6881}
	hash := Hash(NewNodeIDFromHex("E8F3C02AC41F2C7F2A6552F47E85FBDDFC6F8F0F"))
	wire, err := NewTransport(addr, hash)
	if err != nil {
		t.Error(err)
	}
	wire.Conn.Write(PacketHandshake(Hash(NewNodeIDFromHex("E8F3C02AC41F2C7F2A6552F47E85FBDDFC6F8F0F"))))
}
