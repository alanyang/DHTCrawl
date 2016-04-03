package main

import (
	crawl "DHTCrawl"
	"fmt"
	// "strings"
)

func main() {
	dht := crawl.NewDHT(nil)
	dht.HandleHash(func(hash crawl.Hash) bool {
		// fmt.Println(hash.Hex())
		return true
	})
	dht.HandleMetadata(func(info *crawl.MetadataResult) {
		fmt.Println("********************************")
		// fmt.Println(info.Hash.Hex())
		fmt.Println(info.Hash.Magnet())
		fmt.Println(info.Name)
		// if info.Length != 0 {
		// 	fmt.Println(info.Length)
		// }
		// if len(info.Files) != 0 {
		// 	fmt.Println("========FILES==========")
		// 	for _, f := range info.Files {
		// 		fmt.Printf("\t%s (%d)\n", strings.Join(f.Path, "/"), f.Length)
		// 	}
		// 	fmt.Println("=======================")
		// }
		// fmt.Println("********************************\n")
	})
	dht.Run()
}
