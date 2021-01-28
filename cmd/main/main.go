package main

import (
	"flag"
	"log"
	"sort"
	"strings"
	"time"

	dhtcrawl "bitbucket.org/AlanYang/DHTCrawl"
)

const (
	Iso8601Format = "2006-01-02T15:04:05"
)

func Iso8601Now() string {
	return time.Now().Format(Iso8601Format)
}

func MergeMetainfo(r *dhtcrawl.MetadataResult) *dhtcrawl.Metainfo {
	info := new(dhtcrawl.Metainfo)
	if r.UName != "" {
		info.Name = r.UName
	} else {
		info.Name = r.Name
	}
	now := Iso8601Now()
	info.Hex = r.Hash.Hex()
	info.Length = int(r.Length)
	info.Create = now
	info.Last = now
	info.Downloads = 1

	var ext string
	if len(r.Files) != 0 {
		fs := dhtcrawl.Files(r.Files)
		sort.Sort(sort.Reverse(fs))
		mainfile := fs[0]
		ext = dhtcrawl.GetExtension(mainfile.Path[len(mainfile.Path)-1])
	} else {
		ext = dhtcrawl.GetExtension(r.Name)
	}
	info.Type = dhtcrawl.GetMetaType(strings.ToLower(ext))

	info.Files = []*dhtcrawl.MetaFile{}
	for _, f := range r.Files {
		file := new(dhtcrawl.MetaFile)
		if len(f.UPath) != 0 {
			file.Path = strings.Join(f.UPath, "/")
		} else {
			file.Path = strings.Join(f.Path, "/")
		}
		file.Length = int(f.Length)
		info.Files = append(info.Files, file)
	}
	return info
}

func main() {
	elasticUrl := ""
	flag.StringVar(&elasticUrl, "elastic", dhtcrawl.ElasticUrl(), "elasticsearch host")
	flag.Parse()
	log.Println(elasticUrl)
	// ela, err := dhtcrawl.NewElastic(elasticUrl)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	dht := dhtcrawl.NewDHT(nil)
	num := 0

	dht.HandleHash(func(hash dhtcrawl.Hash) bool {
		// _, id, err := ela.GetDocByHex(hash.Hex())
		// if err != nil {
		// 	//no has
		// 	return true
		// }
		// script := "ctx._source.downloads += n;ctx._source.last = l;"
		// params := map[string]interface{}{"n": 1, "l": Iso8601Now()}
		// err = ela.Update(id, script, params)
		// if err != nil {
		// 	log.Printf("Update download error %s\n", err.Error())
		// 	log.Println(id, script, params)
		// }
		// log.Printf("Updated %s done", id)
		return false
	})

	dht.HandleMetadata(func(info *dhtcrawl.MetadataResult) {
		defer func() { num++ }()
		log.Println(num)
		//put data to elasticsearch
		// id, err := ela.Index(MergeMetainfo(info))
		// log.Println(id)
		println(info.String())
		// if err != nil {
		// 	log.Printf("index %s error: %s", info.Hash.Hex(), err.Error())
		// 	return
		// }
	})
	dht.Run()
}
