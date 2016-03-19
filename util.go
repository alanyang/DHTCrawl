package DHTCrawl

import (
	"errors"
	"fmt"
	"math"
	"net"
	"sync/atomic"
)

var (
	tid uint32 = 1
)

func IsValidPort(port int) bool {
	return port > 0 && port < (1<<16)
}

func DecodeNodes(data []byte) ([]*Node, error) {
	if len(data)%26 != 0 {
		return nil, errors.New("Illegal node bytes")
	}
	nodes := []*Node{}
	for j := 0; j < len(data); j = j + 26 {
		if j+26 > len(data) {
			break
		}
		kn := data[j : j+26]
		node := new(Node)
		node.ID = NodeID(kn[0:20])
		port := kn[24:26]
		node.Addr = &net.UDPAddr{IP: net.IP(kn[20:24]), Port: int(port[0])<<8 + int(port[1])}
		if IsValidPort(node.Addr.Port) {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

func GenerateTid() string {
	return fmt.Sprintf("%d", atomic.AddUint32(&tid, 1)%math.MaxInt16)
}
