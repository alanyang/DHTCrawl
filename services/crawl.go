package main

import (
	crawl "DHTCrawl"
	"log"
	"time"
)

const (
	Iso8601Format = "2006-01-02T15:04:05"
)

func Iso8601Now() string {
	return time.Now().Format(Iso8601Format)
}

func main() {
	ela, err := crawl.NewElastic("http://127.0.0.1:9200")
	if err != nil {
		log.Fatal(err)
	}
	// if err = ela.CreateIndex(); err != nil {
	// 	log.Fatal(err, "2")
	// }
	// if err = ela.CreateMapping(); err != nil {
	// 	log.Fatal(err, "3")
	// }
	dht := crawl.NewDHT(nil)
	num := 0
	dht.HandleHash(func(hash crawl.Hash) bool {
		key := []byte(hash)
		_, err := dht.DB.Get(key, nil)
		//not found
		if err != nil {
			err = dht.DB.Put(key, crawl.DBValueUnIndexID, nil)
			if err != nil {
				log.Fatal(err)
				return false
			}
			return true
		}
		return false
	})
	dht.HandleMetadata(func(info *crawl.MetadataResult) {
		log.Println(num)
		println(info.String())
		info.Hex = info.Hash.Hex()
		info.Create = Iso8601Now()
		info.Download = 1
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
