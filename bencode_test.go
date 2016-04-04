package DHTCrawl

import (
	"bytes"
	bencode "github.com/jackpal/bencode-go"
	"testing"
)

func Test_Marshal(t *testing.T) {
	rw := new(bytes.Buffer)
	bencode.Marshal(rw, Data{
		Tid:      "12",
		Type:     "q",
		Query:    "get_peers",
		Argument: Argument{ID: "01234567890123456789", Target: "01234567890123456789", Port: 6632},
	})
	t.Log(string(rw.Bytes()))

	result := new(Data)
	bencode.Unmarshal(rw, result)
	t.Log(result)
}
