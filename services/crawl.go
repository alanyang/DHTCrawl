package main

import (
	crawl "DHTCrawl"
	"github.com/syndtr/goleveldb/leveldb"
	"log"
	"sort"
	"strings"
	"time"
)

const (
	Iso8601Format = "2006-01-02T15:04:05"
)

func Iso8601Now() string {
	return time.Now().Format(Iso8601Format)
}

func MergeMetainfo(r *crawl.MetadataResult) *crawl.Metainfo {
	info := new(crawl.Metainfo)
	if r.UName != "" {
		info.Name = r.UName
	} else {
		info.Name = r.Name
	}
	info.Hex = r.Hash.Hex()
	info.Hex_ = info.Hex
	info.Length = int(r.Length)
	info.Length_ = info.Length
	info.Create = Iso8601Now()
	info.Created = Iso8601Now()
	info.Last = Iso8601Now()
	info.Lasted = Iso8601Now()
	info.Downloads = 1
	info.Downloadeds = 1

	var ext string
	if len(r.Files) != 0 {
		fs := crawl.Files(r.Files)
		sort.Sort(sort.Reverse(fs))
		mainfile := fs[0]
		ext = crawl.GetExtension(mainfile.Path[len(mainfile.Path)-1])
	} else {
		ext = crawl.GetExtension(r.Name)
	}
	info.Type = crawl.GetMetaType(strings.ToLower(ext))

	info.Files = []*crawl.MetaFile{}
	for _, f := range r.Files {
		file := new(crawl.MetaFile)
		if len(f.UPath) != 0 {
			file.Path = strings.Join(f.UPath, "/")
		} else {
			file.Path = strings.Join(f.Path, "/")
		}
		file.Length = int(f.Length)
		file.Length_ = file.Length
		info.Files = append(info.Files, file)
	}
	return info
}

func main() {
	db, err := leveldb.OpenFile(crawl.DBPath(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ela, err := crawl.NewElastic(crawl.ElasticUrl())
	if err != nil {
		log.Fatal(err)
	}
	dht := crawl.NewDHT(nil)
	num := 0
	dht.HandleHash(func(hash crawl.Hash) bool {
		key := []byte(hash)
		d, err := db.Get(key, nil)
		if err != nil {
			err = db.Put(key, crawl.DBValueUnIndexID, nil)
			if err != nil {
				log.Fatal(err)
				return false
			}
			return true
		} else {
			id := string(d)
			//undownload
			if id == string(crawl.DBValueUnIndexID) {
				return true
			}
			script := "ctx._source._downloads += n;ctx._source._last = l;ctx._source.downloads += n;ctx._source.last = l;"
			params := map[string]interface{}{"n": 1, "l": Iso8601Now()}
			err := ela.Update(id, script, params)
			if err != nil {
				log.Printf("Update download error %s\n", err.Error())
				log.Println(id, script, params)
			}
			log.Println("Update success")
		}

		return false
	})

	dht.HandleMetadata(func(info *crawl.MetadataResult) {
		defer func() { num++ }()
		log.Println(num)
		println(info.String())
		//put data to elasticsearch
		id, err := ela.Index(MergeMetainfo(info))
		if err != nil {
			log.Printf("index %s error: %s", info.Hash.Hex(), err.Error())
			return
		}
		//put elasticsearch document id to leveldb
		key := []byte(info.Hash)
		err = db.Put(key, []byte(id), nil)
		if err != nil {
			log.Fatal(err)
		}
	})
	dht.Run()
}
