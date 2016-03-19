package DHTCrawl

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"time"
)

const (
	MAX_COUNT = 500
)

type (
	NodeID []byte

	Hash NodeID

	Node struct {
		ID   NodeID
		Addr *net.UDPAddr
	}

	Table struct {
		Nodes []*Node
		Self  NodeID
	}
)

func NewNodeID() NodeID {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := sha1.New()
	io.WriteString(hash, time.Now().String())
	io.WriteString(hash, string(random.Int()))
	io.WriteString(hash, "AmyLee+AlanYang!")
	return hash.Sum(nil)
}

func NewNodeIDFromHex(hex string) NodeID {
	if len(hex) != 40 {
		return nil
	}
	id := make([]byte, 20)
	j := 0
	for i := 0; i < len(hex); i += 2 {
		n1, _ := strconv.ParseInt(hex[i:i+1], 16, 8)
		n2, _ := strconv.ParseInt(hex[i+1:i+2], 16, 8)
		id[j] = byte((n1 << 4) + n2)
		j++
	}
	return id
}

func (n NodeID) String() string {
	return bytes.NewBuffer(n).String()
}

func (n NodeID) Hex() string {
	return fmt.Sprintf("%X", n)
}

func (n NodeID) Neighbor() NodeID {
	return append(n[:12], NewNodeID()[:8]...)
}

func NewNode() *Node {
	return &Node{ID: NewNodeID()}
}

func NewTable() *Table {
	return &Table{Nodes: []*Node{}, Self: NewNodeID()}
}

func (t *Table) Add(node *Node) {
	t.Nodes = append(t.Nodes)
}
