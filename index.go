package DHTCrawl

import (
	"errors"
	"gopkg.in/olivere/elastic.v3"
)

const (
	Index   = "bittorrent"
	Type    = "metainfo"
	Mapping = `
	{
		"metainfo":{
			"_all": {
            	"analyzer": "smartcn",
            	"search_analyzer": "smartcn",
            	"store": "true"
			},
			"properties":{
				"name":{
					"type":"string",
					"boost": "2.0"
				},
				"hash": {
					"type": "string",
					"index": "not_analyzed"
				},
				"hex": {
					"type": "string",
					"index": "not_analyzed"
				},
				"length": {
					"type": "long",
					"index": "no"
				},
				"create": {
					"type": "date",
					"index": "no"
				},
				"files":{
                	"properties":{
                    	"path":{"type":"string", "boost":"1.5"},
                    	"upath":{"type":"string", "boost":"1.5"},
                    	"length":{"type":"long", "index": "no"}
                	}
            	}
			}
		}
	}
	`
)

type (
	Elastic struct {
		Url  string
		Conn *elastic.Client
	}
)

func NewElastic(url string) (*Elastic, error) {
	ela := new(Elastic)
	ela.Url = url
	conn, err := elastic.NewClient(elastic.SetURL(url))
	if err != nil {
		return nil, err
	}
	ela.Conn = conn
	return ela, nil
}

func (e *Elastic) IndexExists() (ok bool, err error) {
	ok, err = e.Conn.IndexExists(Index).Do()
	return
}

func (e *Elastic) DeleteIndex() (err error) {
	resp, err := e.Conn.DeleteIndex(Index).Do()
	if err != nil {
		return err
	}
	if !resp.Acknowledged {
		return errors.New("Delete index fail")
	}
	return nil
}

func (e *Elastic) CreateIndex() error {
	exists, err := e.IndexExists()
	if exists {
		return nil
	}
	result, err := e.Conn.CreateIndex(Index).Do()
	if err != nil {
		return err
	}
	if !result.Acknowledged {
		return errors.New("create index fail")
	}
	return nil
}

func (e *Elastic) CreateMapping() error {
	resp, err := e.Conn.PutMapping().Index(Index).Type(Type).BodyString(Mapping).Do()
	if err != nil {
		return err
	}
	if resp == nil || !resp.Acknowledged {
		return errors.New("Mapping fail")
	}
	return nil
}

func (e *Elastic) GetMapping() (interface{}, error) {
	resp, err := e.Conn.GetMapping().Index(Index).Type(Type).Do()
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("Get mapping fail")
	}
	prop, ok := resp[Index]
	if !ok {
		return nil, errors.New("Get mapping fail")
	}
	return prop, nil
}

func (e *Elastic) Index(meta *MetadataResult) (string, error) {
	resp, err := e.Conn.Index().Index(Index).Type(Type).BodyJson(*meta).Do()
	if err != nil {
		return "-1", err
	}
	return resp.Id, nil
}

func (e *Elastic) Update(id, script string, params map[string]interface{}) error {
	s := elastic.NewScript(script).Params(params).Lang("groovy")
	resp, err := e.Conn.Update().Index(Index).Type(Type).Id(id).Script(s).ScriptedUpsert(false).Do()
	if err != nil {
		return err
	}
	if resp.Id == "" {
		return errors.New("Update fail")
	}
	return nil
}
