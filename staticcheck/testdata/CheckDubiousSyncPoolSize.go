package pkg

import "sync"

type T1 struct {
	x int
}

type T2 struct {
	x int
	y int
}

func fn() {
	s := []int{}

	v := sync.Pool{}
	v.Put(s) // MATCH /argument should be one word large or less/
	v.Put(&s)
	v.Put(T1{})
	v.Put(T2{}) // MATCH /argument should be one word large or less/

	p := &sync.Pool{}
	p.Put(s) // MATCH /argument should be one word large or less/
	p.Put(&s)

	var i interface{}
	p.Put(i)
}
