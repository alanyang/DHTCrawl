package DHTCrawl

import (
	"github.com/zeebo/bencode"
	"testing"
)

func Test_Bencode(t *testing.T) {
	d := map[string]interface{}{
		"t": GenerateTid(),
		"y": TYPE_QUERY,
		"q": OP_FIND_NODE,
		"a": map[string]string{
			"id":     "6a60d75573b3e30e052b1168c902c623a6a21ba1",
			"target": "33ee6e6ae87f24e3e9a0aa308c57cb289a927a04",
		},
	}
	d1, _ := bencode.EncodeString(d)
	d2, _ := bencode.EncodeBytes(d)
	if d1 != string(d2) {
		t.Error("Not equal")
	}
	t.Log(d1)
	t.Log(string(d2))
}
