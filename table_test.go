package DHTCrawl

import (
	"testing"
)

func Test_SpyID(t *testing.T) {
	a := NewNodeID()
	t.Log(a.Hex())
	t.Log(a.Neighbor().Hex())
}
