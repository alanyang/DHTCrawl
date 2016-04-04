package main

import (
	crawl "DHTCrawl"
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"strings"
)

func main() {
	dht := crawl.NewDHT(nil)
	num := 0
	dht.HandleHash(func(hash crawl.Hash) bool {
		return true
		key := []byte(hash)
		_, err := dht.DB.Get(key, nil)
		//not found
		if err != nil {
			err = dht.DB.Put(key, crawl.DBValueUnDownload, nil)
			if err != nil {
				log.Fatal(err)
				return false
			}
			return true
		}
		return false
	})
	dht.HandleMetadata(func(info *crawl.MetadataResult) {
		num++
		log.Println(num)
		fmt.Println("********************************")
		fmt.Println(info.Hash.Hex())
		fmt.Println(info.Hash.Magnet())
		fmt.Println(info.Name)
		if info.Length != 0 {
			fmt.Println(info.Length)
		}
		if len(info.Files) != 0 {
			fmt.Println("========FILES==========")
			for _, f := range info.Files {
				fmt.Printf("\t%s (%d)\n", strings.Join(f.Path, "/"), f.Length)
			}
			fmt.Println("=======================")
		}
		fmt.Println("********************************\n")
		key := []byte(info.Hash)
		buf := new(bytes.Buffer)
		en := gob.NewEncoder(buf)
		err := en.Encode(*info)
		if err != nil {
			log.Fatal(err)
		}
		err = dht.DB.Put(key, buf.Bytes(), nil)
		if err != nil {
			log.Fatal(err)
		}
	})
	dht.Run()
}
