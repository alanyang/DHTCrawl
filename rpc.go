package DHTCrawl

import (
	"bytes"
	"errors"
	"github.com/zeebo/bencode"
	"net"
	// "reflect"
)

const (
	TYPE_QUERY    = "q"
	TYPE_RESPONSE = "r"
	TYPE_ERROR    = "e"

	OP_FIND_NODE     = "find_node"
	OP_PING          = "ping"
	OP_GET_PEERS     = "get_peers"
	OP_ANNOUNCE_PEER = "announce_peer"
)

type (
	Result struct {
		Cmd     string
		Hash    Hash
		ID      NodeID
		UDPAddr *net.UDPAddr
		TCPAddr *net.TCPAddr
		Token   string
		Nodes   []*Node
		Tid     string
	}

	RPC struct {
	}
)

//query
func PacketFindNode(id, target NodeID) []byte {
	d := map[string]interface{}{
		"t": GenerateTid(),
		"y": TYPE_QUERY,
		"q": OP_FIND_NODE,
		"a": map[string]string{
			"id":     id.String(),
			"target": target.String(),
		},
	}
	b, _ := bencode.EncodeBytes(d)
	return b
}

//response
//id is self id
func PacketGetPeers(hash Hash, id NodeID, nodes []byte, token, tid string) []byte {
	d := map[string]interface{}{
		"t": tid,
		"y": TYPE_RESPONSE,
		"r": map[string]string{
			"id":    NodeID(hash).Neighbor().String(),
			"nodes": bytes.NewBuffer(nodes).String(),
			"token": token,
		},
	}
	b, _ := bencode.EncodeBytes(d)
	return b
}

func PacketAnnucePeer(hash Hash, id NodeID, tid string) []byte {
	d := map[string]interface{}{
		"t": tid,
		"y": TYPE_RESPONSE,
		"r": map[string]string{"id": NodeID(hash).Neighbor().String()},
	}
	b, _ := bencode.EncodeBytes(d)
	return b
}

func PacketPong(id NodeID, tid string) []byte {
	d := map[string]interface{}{
		"t": tid,
		"y": TYPE_RESPONSE,
		"r": map[string]string{"id": id.Neighbor().String()},
	}
	b, _ := bencode.EncodeBytes(d)
	return b
}

func NewRPC() *RPC {
	return &RPC{}
}

func (r *RPC) HandleGetPeers(args map[string]interface{}) Hash {
	if hash, ok := args["info_hash"].(string); ok {
		return Hash(hash)
	}
	return ""
}

//info hash, tcp port, token
func (r *RPC) HandleAnnoucePeer(args map[string]interface{}) (hash Hash, port int64, token string) {
	if h, ok := args["info_hash"].(string); ok {
		hash = Hash(h)
	}
	port, _ = args["port"].(int64)
	token, _ = args["token"].(string)

	//if implied_port not equal 0, then use same udp port for tcp
	if implied, ok := args["implied_port"].(int64); ok && implied != int64(0) {
		port = int64(-1)
	}

	return
}

func (r *RPC) HandleFindNode(resp map[string]interface{}) (ns []*Node) {
	if b, ok := resp["nodes"].(string); ok {
		if nodes, err := DecodeNodes([]byte(b)); err == nil {
			for _, node := range nodes {
				ns = append(ns, node)
			}
		}
	}
	return
}

func (r *RPC) handlePing(args map[string]interface{}) NodeID {
	if id, ok := args["id"].(string); ok {
		return NodeID([]byte(id))
	}
	return nil
}

func (r *RPC) parse(data []byte, addr *net.UDPAddr) (*Result, error) {
	v := make(map[string]interface{})

	if err := bencode.DecodeBytes(data, &v); err != nil {
		return nil, err
	}

	t, ok := v["t"].(string)
	if !ok {
		return nil, errors.New("Invalid protocol field 't'")
	}

	y, ok := v["y"].(string)
	if !ok {
		return nil, errors.New("Invalid protocol field 'y'")
	}

	switch y {
	case TYPE_QUERY:
		q, ok := v["q"].(string)
		if !ok {
			return nil, errors.New("Invalid protocol field 'q'")
		}
		a, ok := v["a"].(map[string]interface{})
		if !ok {
			return nil, errors.New("Invalid protocol field 'a'")
		}
		switch q {
		case OP_GET_PEERS:
			hash := r.HandleGetPeers(a)
			if string(hash) != "" {
				return &Result{Cmd: OP_GET_PEERS, UDPAddr: addr, Hash: hash, Tid: t}, nil
			}
		case OP_ANNOUNCE_PEER:
			hash, port, token := r.HandleAnnoucePeer(a)
			if port == int64(-1) {
				port = int64(addr.Port)
			}
			if string(hash) != "" && IsValidPort(int(port)) {
				tcpAddr := &net.TCPAddr{IP: addr.IP, Port: int(port)}
				return &Result{
					Cmd:     OP_ANNOUNCE_PEER,
					UDPAddr: addr,
					Hash:    hash,
					TCPAddr: tcpAddr,
					Token:   token,
					Tid:     t,
				}, nil
			}
		case OP_PING:
			if id := r.handlePing(a); id != nil {
				return &Result{
					Cmd:     OP_PING,
					UDPAddr: addr,
					Tid:     t,
					ID:      id,
				}, nil
			}
		default:
		}
	case TYPE_RESPONSE:
		//only has find_node response
		if a, ok := v["r"].(map[string]interface{}); ok {
			return &Result{Cmd: OP_FIND_NODE, Nodes: r.HandleFindNode(a), Tid: t}, nil
		}
	case TYPE_ERROR:
	default:
	}
	return nil, errors.New("Invalid protocol")
}
