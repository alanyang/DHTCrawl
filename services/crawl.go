package main

import (
	crawl "DHTCrawl"
	// "log"
)

func main() {
	dht := crawl.NewDHT(nil)
	dht.Handle(func(hash crawl.Hash) bool {
		// log.Printf("%X", []byte(hash))
		// println(hash.Hex())
		return true
	})
	dht.Run()
}
