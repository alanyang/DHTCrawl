package main

import (
	"DHTCrawl"
	"github.com/valyala/gorpc"
	"log"
)

const (
	SERVICE_PORT = ":1128"
)

func main() {
	DHTCrawl.TorrentClient = DHTCrawl.NewTorrentClient()

	s := gorpc.NewTCPServer(SERVICE_PORT, DHTCrawl.Dispatcher.NewHandlerFunc())

	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}
