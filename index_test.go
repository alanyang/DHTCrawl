package DHTCrawl

import (
	"gopkg.in/olivere/elastic.v3"
	"testing"
	"time"
)

func Test_Update(t *testing.T) {
	ela, _ := NewElastic(ElasticUrl())
	script := `ctx._source.download += n`
	params := map[string]interface{}{"n": time.Now().Format("2006-01-02T15:04:05")}
	if err := ela.Update("AVPqKBiZ-ADMSk-IEqoD", script, params); err != nil {
		t.Error(err.Error())
	}
}

func Test_Get(t *testing.T) {
	client, _ := elastic.NewClient(elastic.SetURL(ElasticUrl()))
	query := elastic.NewMatchQuery("hex", "951B8DA3AB58B22D759F5FBA6D22FFA9E6242CED")
	resp, err := client.Search().Index(Index).Type(Type).Query(query).Do()
	t.Log(resp)
	if err != nil {
		t.Error(err)
	}
}
