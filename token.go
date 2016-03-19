package DHTCrawl

import (
	"fmt"
	"math/rand"
	"time"
)

type Token struct {
	Value    string
	prev     string
	Duration time.Duration // Duration * minute
}

func NewToken(duration int) *Token {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	t := Token{fmt.Sprintf("%x", r.Int()), "", time.Duration(duration)}
	go t.refresh()
	return &t
}

func (t *Token) refresh() {
	for {
		time.Sleep(time.Minute * t.Duration)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		t.prev = t.Value
		t.Value = fmt.Sprintf("%x", r.Int())
	}
}

func (t *Token) IsValid(v string) (b bool) {
	return v == t.Value || v == t.prev
}
