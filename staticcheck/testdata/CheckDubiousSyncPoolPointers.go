package pkg

import "sync"

func fn() {
	s := []int{}

	v := sync.Pool{}
	v.Put(s) // MATCH /non-pointer type/
	v.Put(&s)

	p := &sync.Pool{}
	p.Put(s) // MATCH /non-pointer type/
	p.Put(&s)
}
