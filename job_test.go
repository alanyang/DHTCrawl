package DHTCrawl

import "testing"

func Test_Set(t *testing.T) {
	s := NewSet()
	s.Set("111")
	t.Log(s.String())
	if !s.Has("111") {
		t.Error("Set has error")
	}
	s.Delete("111")
	t.Log(s.String())
	if s.Has("111") {
		t.Error("Delete has error")
	}
}
