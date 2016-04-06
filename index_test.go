package DHTCrawl

import (
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
