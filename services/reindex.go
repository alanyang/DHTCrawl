package main

import (
	crawl "DHTCrawl"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"gopkg.in/olivere/elastic.v3"
	"log"
	"time"
)

func main() {
	db, err := leveldb.OpenFile(crawl.DBPath(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ela, err := crawl.NewElastic(crawl.ElasticUrl())
	if err != nil {
		log.Fatal(err)
	}

	conn := ela.Conn
	numSuccess := 0
	numFailed := 0

	it := db.NewIterator(nil, nil)
	script := elastic.NewScript(`ctx._source._downloads = ctx._source.downloads;ctx._source._last = ctx._source.last; ctx._source._create = ctx._source.create;ctx._source._hex = ctx._source.hex;`)
	for it.Next() {
		id := string(it.Value())
		if id != string(crawl.DBValueUnIndexID) {
			resp, err := conn.Update().Index(crawl.Index).Type(crawl.Type).Id(id).Script(script).ScriptedUpsert(false).Do()
			if err != nil {
				fmt.Printf("update %s error %s\n", id, err.Error())
				numFailed++
				continue
			}
			if resp.Id == "" {
				fmt.Printf("update failed\n")
				numFailed++
				continue
			}
			numSuccess++
			fmt.Printf("update document %s successful[%d]\n", id, numSuccess)
			time.Sleep(time.Millisecond * 2)
		}
	}
	fmt.Printf("Updated document %d, failed %d\n", numSuccess, numFailed)
}
