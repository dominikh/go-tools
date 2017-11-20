package pkg

import (
	"sync"
	"unsafe"
)

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
	v.Put(s) // MATCH /argument should be pointer-like/
	v.Put(&s)
	v.Put(T1{}) // MATCH /argument should be pointer-like/
	v.Put(T2{}) // MATCH /argument should be pointer-like/

	p := &sync.Pool{}
	p.Put(s) // MATCH /argument should be pointer-like/
	p.Put(&s)

	var i interface{}
	p.Put(i)

	var up unsafe.Pointer
	p.Put(up)

	var basic int
	p.Put(basic) // MATCH /argument should be pointer-like/
}
