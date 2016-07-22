package pkg

import "sync"

func fn() {
	s := []int{}

	v := sync.Pool{}
	v.Put(s) // MATCH /Non-pointer type /
	v.Put(&s)

	p := &sync.Pool{}
	p.Put(s) // MATCH /Non-pointer type /
	p.Put(&s)
}
