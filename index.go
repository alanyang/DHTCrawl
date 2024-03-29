package DHTCrawl

import (
	"encoding/json"
	"errors"
	"gopkg.in/olivere/elastic.v3"
	// "log"
)

const (
	Index   = "bittorrent2"
	Type    = "metainfo"
	Mapping = `
	{
		"metainfo":{
			"_all": {
            	"store": "true"
			},
			"properties":{
				"name": {
					"type":"string",
					"boost": "2.0",
					"analyzer": "ik_max_word"
				},
				"hex": {
					"type": "string"
				},
				"length": {
					"type": "long"
				},
				"type": {
					"type": "long"
				},
				"create": {
					"type": "date"
				},
				"downloads": {
					"type": "long"
				},
				"last": {
					"type": "date"
				},
				"files":{
					"type" : "nested",
					"include_in_parent": "true",
					"properties": {
						"path": {"type": "string", "boost": "1.5", "analyzer": "ik_max_word"},
						"length": {"type": "long"}
					}
            	}
			}
		}
	}
	`
)

/*
"properties":{
                    	"path":{"type":"string", "boost":"1.5"},
                    	"upath":{"type":"string", "boost":"1.5"},
                    	"length":{"type":"long", "index": "no"}
          }
*/
type (
	Elastic struct {
		Url  string
		Conn *elastic.Client
	}

	MetaFile struct {
		Path   string `json:"path,omitempty"`
		Length int    `json:"length,omitempty"`
	}

	Metainfo struct {
		Name      string      `json:"name,omitempty"`
		Hex       string      `json:"hex,omitempty"`
		Length    int         `json:"length,omitempty"`
		Create    string      `json:"create,omitempty"`
		Last      string      `json:"last,omitempty"`      //last download
		Downloads int         `json:"downloads,omitempty"` //downloads count
		Type      int         `json:"type,omitempty"`
		Files     []*MetaFile `json:"files,omitempty"`
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

func (e *Elastic) Index(meta *Metainfo) (string, error) {
	resp, err := e.Conn.Index().Index(Index).Type(Type).BodyJson(*meta).Do()
	if err != nil {
		return "-1", err
	}
	return resp.Id, nil
}

func (e *Elastic) Update(id, script string, params map[string]interface{}) error {
	s := elastic.NewScriptInline(script).Params(params)
	req := e.Conn.Update().Index(Index).Type(Type).Id(id).Script(s).ScriptedUpsert(false)
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp.Id == "" {
		return errors.New("Update fail")
	}
	return nil
}

func (e *Elastic) GetDocByHex(hex string) (*Metainfo, string, error) {
	query := elastic.NewTermQuery("hex", hex)
	resp, err := e.Conn.Search().Index(Index).Type(Type).Query(query).Do()
	if err != nil {
		return nil, "", err
	}
	for _, hit := range resp.Hits.Hits {
		item := new(Metainfo)
		err := json.Unmarshal(*hit.Source, item)
		if err != nil {
			return nil, "", err
		}
		return item, hit.Id, nil
	}

	return nil, "", errors.New("Not has the document")
}
