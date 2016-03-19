package main

import (
	crawl "DHTCrawl"
	"log"
	"net"
)

func main() {
	dht := crawl.NewDHT()
	dht.FoundNodes = FoundNodes
	dht.UnsureHash = func(hash crawl.Hash, _ *net.UDPAddr) {
		log.Printf("Got a unsure hash magnet:?xt=urn:btih:%x", []byte(hash))
	}
	dht.EnsureHash = func(hash crawl.Hash, _ *net.UDPAddr, _ *net.TCPAddr) {
		log.Printf("Got a ensure hash magnet:?xt=urn:btih:%x", []byte(hash))
	}
	dht.Run()
}

func FoundNodes(ns []*crawl.Node, _ *net.UDPAddr) {
	// log.Printf("Found nodes %d", len(ns))
}
