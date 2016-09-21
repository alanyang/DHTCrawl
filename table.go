package DHTCrawl

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

type (
	EachHandler func(*Node, int)

	NodeID []byte

	Hash string

	Node struct {
		ID   NodeID
		Addr *net.UDPAddr
	}

	Table struct {
		Nodes []*Node
		Last  []*Node
		Self  NodeID
		Mutex *sync.RWMutex
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

func (n NodeID) Neighbor(target NodeID) NodeID {
	return append(n[:8], target[8:]...)
}

func (h Hash) Hex() string {
	return fmt.Sprintf("%X", []byte(h))
}

func (h Hash) Magnet() string {
	return fmt.Sprintf("magnet:?xt=urn:btih:%X", []byte(h))
}

func NewNode() *Node {
	return &Node{ID: NewNodeID()}
}

func NewTable() *Table {
	return &Table{Nodes: []*Node{}, Self: NewNodeID(), Mutex: new(sync.RWMutex)}
}

func (t *Table) Flush() {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.Nodes = []*Node{}
}

func (t *Table) Add(node *Node) {
	t.Mutex.Lock()
	defer t.Mutex.Unlock()
	t.Nodes = append(t.Nodes, node)
	t.Last = append(t.Last, node)
	if len(t.Last) > 8 {
		t.Last = t.Last[1:]
	}
}

func (t *Table) Each(handler EachHandler) {
	t.Mutex.RLock()
	nodes := t.Nodes[:]
	t.Mutex.RUnlock()
	for i, node := range nodes {
		handler(node, i)
	}
}
