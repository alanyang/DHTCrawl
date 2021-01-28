package DHTCrawl

import (
	"fmt"
	"net"
	"sync"
)

type (
	Job struct {
		Hash Hash
		Addr *net.TCPAddr
	}
	WireJob struct {
		Size       int
		jobChan    chan *Job
		resultChan chan *MetadataResult
		Result     chan *MetadataResult
		stopChan   chan int
		worker     []*Wire
		started    *Set
		jobsQueue  *Set
	}

	Set struct {
		data map[interface{}]struct{}
		mu   *sync.RWMutex
	}
)

func NewSet() *Set {
	return &Set{data: make(map[interface{}]struct{}), mu: new(sync.RWMutex)}
}

func (s *Set) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := []interface{}{}
	for key := range s.data {
		keys = append(keys, key)
	}
	return fmt.Sprintf("Set(%v)", keys)
}

func (s *Set) Set(v interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[v] = struct{}{}
}

func (s *Set) Has(v interface{}) (ok bool) {
	s.mu.RLock()
	s.mu.RUnlock()
	_, ok = s.data[v]
	return
}

func (s *Set) Pop() (val interface{}) {
	s.mu.RLock()
	s.mu.RUnlock()
	for _, v := range s.data {
		val = v
	}
	if val != nil {
		s.Delete(val)
	}
	return
}

func (s *Set) Delete(v interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, v)
}

func NewJob(hash Hash, addr *net.TCPAddr) *Job {
	return &Job{hash, addr}
}

func NewWireJob(size int) *WireJob {
	wj := &WireJob{
		Size:       size,
		Result:     make(chan *MetadataResult),
		resultChan: make(chan *MetadataResult),
		jobChan:    make(chan *Job),
		worker:     []*Wire{},
		started:    NewSet(),
		jobsQueue:  NewSet(),
	}
	for i := 0; i < size; i++ {
		wire := NewWire(wj.resultChan)
		wj.worker = append(wj.worker, wire)
	}
	go func() {
		for {
			wj.handleResult(<-wj.resultChan)
		}
	}()
	go func() {
		for {
			wj.addJob(<-wj.jobChan)
		}
	}()
	return wj
}

func (j *WireJob) handleResult(r *MetadataResult) {
	j.started.Delete(r.Hash)
	if r.Name != "" {
		j.Result <- r
	}
	if v := j.jobsQueue.Pop(); v != nil {
		if job, ok := v.(*Job); ok {
			j.addJob(job)
		}
	}
}

func (j *WireJob) addJob(job *Job) {
	if !j.started.Has(job.Hash) {
		for _, w := range j.worker {
			if w.IsIdle() {
				j.started.Set(job.Hash)
				w.Job <- job
				return
			}
		}
		fmt.Printf("Not has idle worker %s[%s]\n", job.Hash.Hex(), job.Addr.String())
		j.jobsQueue.Set(job)
	}
}

func (j *WireJob) Stop() {
	j.stopChan <- 0
}

func (j *WireJob) Add(job *Job) {
	j.jobChan <- job
}
