package DHTCrawl

import (
	"fmt"
	"github.com/zeebo/bencode"
	"net/http"
)

const (
	Url = "http://bt.box.n0808.com"
)

type MetaData struct {
	Hash     Hash
	MetaInfo *MetadataResult `bencode:"info"`
}

func HttpDownload(hash Hash) (*MetadataResult, error) {
	hex := hash.Hex()
	url := fmt.Sprintf("%s/%s/%s/%s.torrent", Url, hex[:2], hex[38:], hex)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", Url)
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	dec := bencode.NewDecoder(resp.Body)
	metadata := new(MetaData)
	err = dec.Decode(metadata)
	if err != nil {
		return nil, err
	}
	metadata.MetaInfo.Hash = hash
	return metadata.MetaInfo, nil
}
