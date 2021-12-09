package main

import (
	"log"

	dhtcrawl "bitbucket.org/AlanYang/DHTCrawl"
)

func main() {

	dht := dhtcrawl.NewDHT(nil)

	dht.HandleHash(func(hash dhtcrawl.Hash) bool {
		log.Println(hash)
		return false
	})

	dht.HandleMetadata(func(info *dhtcrawl.MetadataResult) {
		println(info.String())
	})
	dht.Run()
}
