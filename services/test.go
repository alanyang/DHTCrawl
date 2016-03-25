package main

import (
	"log"
	"sync"
	"time"
)

func main() {
	ages := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	m := &sync.Mutex{}
	go func() {
		for i := 10; i < 45; i++ {
			time.Sleep(time.Millisecond * 500)
			m.Lock()
			ages = append(ages, i)
			m.Unlock()
			log.Println(ages)
		}
	}()
	for _, a := range ages {
		log.Println(a)
		time.Sleep(time.Second * 1)
	}
	m.Lock()
	ages = []int{}
	m.Unlock()
	time.Sleep(time.Second * 10)
	for _, a := range ages {
		log.Println(a)
		time.Sleep(time.Second * 1)
	}
	log.Println(ages)
}
