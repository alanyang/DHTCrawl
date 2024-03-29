package DHTCrawl

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
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

func ConvertByteStream(nodes []*Node) []byte {
	buf := bytes.NewBuffer(nil)
	for _, v := range nodes {
		convertNodeInfo(buf, v)
	}
	return buf.Bytes()
}

func convertNodeInfo(buf *bytes.Buffer, v *Node) {
	buf.Write([]byte(v.ID))
	convertIPPort(buf, []byte(v.Addr.IP), v.Addr.Port)
}
func convertIPPort(buf *bytes.Buffer, ip net.IP, port int) {
	buf.Write(ip.To4())
	buf.WriteByte(byte((port & 0xFF00) >> 8))
	buf.WriteByte(byte(port & 0xFF))
}

func GenerateTid() string {
	return fmt.Sprintf("%d", atomic.AddUint32(&tid, 1)%math.MaxInt16)
}
func StringToIPBytes(ip string) (b []byte) {
	s := strings.Split(ip, ".")
	for _, i := range s {
		p, _ := strconv.Atoi(i)
		b = append(b, byte(p))
	}
	return
}
