package main

import (
	crawl "DHTCrawl"
	"log"
	"sort"
	"strings"
	"time"
)

const (
	Iso8601Format = "2006-01-02T15:04:05"
)

func Iso8601Now() string {
	return time.Now().Format(Iso8601Format)
}

func main() {
	ela, err := crawl.NewElastic(crawl.ElasticUrl())
	if err != nil {
		log.Fatal(err)
	}
	dht := crawl.NewDHT(nil)
	num := 0
	dht.HandleHash(func(hash crawl.Hash) bool {
		key := []byte(hash)
		d, err := dht.DB.Get(key, nil)
		if err != nil {
			err = dht.DB.Put(key, crawl.DBValueUnIndexID, nil)
			if err != nil {
				log.Fatal(err)
				return false
			}
			return true
		} else {
			id := string(d)
			//undownload
			if id == string(crawl.DBValueUnIndexID) {
				return true
			}
			err := ela.Update(
				id,
				"ctx._source.download += n",
				map[string]interface{}{"n": Iso8601Now()})
			if err != nil {
				log.Printf("Update download error %s\n", err.Error())
			}
		}

		return false
	})
	dht.HandleMetadata(func(info *crawl.MetadataResult) {
		defer func() { num++ }()
		log.Println(num)
		println(info.String())
		info.Hex = info.Hash.Hex()
		info.Create = Iso8601Now()
		info.Download = []string{info.Create}
		var ext string
		if len(info.Files) != 0 {
			fs := crawl.Files(info.Files)
			sort.Sort(sort.Reverse(fs))
			mainfile := fs[0]
			ext = crawl.GetExtension(mainfile.Path[len(mainfile.Path)-1])
		} else {
			ext = crawl.GetExtension(info.Name)
		}
		info.Type = crawl.GetMetaType(strings.ToLower(ext))
		id, err := ela.Index(info)
		if err != nil {
			log.Printf("index %s error: %s", info.Hash.Hex(), err.Error())
			return
		}
		key := []byte(info.Hash)
		err = dht.DB.Put(key, []byte(id), nil)
		if err != nil {
			log.Fatal(err)
		}
	})
	dht.Run()
}
