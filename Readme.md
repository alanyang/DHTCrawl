# DHT Resource crawl

develop for [engiy.com](http://engiy.com)(A bittorrent resource search engine) [shutdown]
Implements [DHT protocol](http://www.bittorrent.org/beps/bep_0005.html)(http://www.bittorrent.org/beps/bep_0006.html)

## requirement
go 1.10+


## install
```
go get https://github.com/alanyang/dhtspider
```


## Useage
```go
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
		log.Println(info.String())
	})
	dht.Run()
}
```


