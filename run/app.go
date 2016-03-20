package main

import (
	crawl "DHTCrawl"
	"log"
)

func main() {
	dht := crawl.NewDHT()
	dht.Handle(func(hash crawl.Hash) bool {
		log.Printf("%X", []byte(hash))
		return true
	})
	dht.Run()
}
