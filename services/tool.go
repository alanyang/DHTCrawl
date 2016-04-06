package main

import (
	. "DHTCrawl"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
)

func main() {
	app := cli.NewApp()
	elasticUrl := ElasticUrl()
	databasePath := DBPath()
	var hash string
	app.Commands = []cli.Command{
		{
			Name:    "init",
			Aliases: []string{"i"},
			Usage:   "initialize crawl and webapp",
			Action: func(c *cli.Context) {
				ela, err := NewElastic(elasticUrl)
				if err != nil {
					fmt.Printf("Create Elastic client error :%s\n", err.Error())
					return
				}
				if err := ela.CreateIndex(); err != nil {
					fmt.Printf("Create index error :%s\n", err.Error())
					return
				}
				fmt.Println("Created index")
				if err := ela.CreateMapping(); err != nil {
					fmt.Printf("Create mapping error:%s\n", err.Error())
					return
				}
				fmt.Println("Created mapping")
			},
		},
		{
			Name:    "flush",
			Aliases: []string{"f"},
			Usage:   "flush all database and index",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "levelDB",
					Value:       databasePath,
					Usage:       "Path of leveldb for crawler",
					Destination: &databasePath,
				},
				cli.StringFlag{
					Name:        "elastic",
					Value:       elasticUrl,
					Usage:       "Url of elasticsearch",
					Destination: &elasticUrl,
				},
			},
			Action: func(c *cli.Context) {
				err := RemoveAll(databasePath)
				if err != nil {
					fmt.Printf("Delete database error: %s\n", err.Error())
				} else {
					fmt.Println("Deleted database")
				}
				ela, err := NewElastic(elasticUrl)
				if err != nil {
					fmt.Printf("Create Elastic client error :%s\n", err.Error())
					return
				}
				err = ela.DeleteIndex()
				if err != nil {
					fmt.Printf("Delete elastic index error: %s\n", err.Error())
				} else {
					fmt.Println("Deleted elastic index")
				}
			},
		},
		{
			Name:  "ID",
			Usage: "Get a elasticID from info hash",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "hash",
					Usage:       "info hash hex",
					Destination: &hash,
				},
			},
			Action: func(c *cli.Context) {
				db, err := leveldb.OpenFile(databasePath, nil)
				if err != nil {
					fmt.Printf("Open leveldb error: %s\n", err.Error())
					return
				}
				key := NewNodeIDFromHex(hash)
				d, err := db.Get(key, nil)
				if err != nil {
					fmt.Printf("Not found elastic ID %s\n", err.Error())
					return
				}
				fmt.Printf("Elastic ID of Info hash %s is %s\n", hash, string(d))
			},
		},
	}
	app.Run(os.Args)
}
