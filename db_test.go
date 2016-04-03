package DHTCrawl

import (
	"github.com/syndtr/goleveldb/leveldb"
	"runtime"
	"testing"
)

func Test_DB(t *testing.T) {
	p := ""
	switch runtime.GOOS {
	case "linux":
		p = "/root/develop/db"
	case "darwin":
		p = "/Users/alanyang/Develop/db"
	}
	t.Log(p)
	db, err := leveldb.OpenFile(p, nil)
	if err != nil {
		t.Error(err.Error())
		return
	}
	db.Put([]byte("111"), []byte("111"), nil)
	d, err := db.Get([]byte("111"), nil)
	t.Log(d)
	if err != nil {
		t.Log(err.Error())
	}
}
