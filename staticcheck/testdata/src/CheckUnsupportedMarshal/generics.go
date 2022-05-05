//go:build go1.18

package pkg

import (
	"encoding/json"
	"encoding/xml"
)

type LMap[K comparable, V any] struct {
	M1 map[K]V
	M2 map[K]chan int
}

func (lm *LMap[K, V]) MarshalJSON() {
	json.Marshal(lm.M1)
	json.Marshal(lm.M2) //@ diag(`unsupported type`)
}

func recursiveGeneric() {
	// don't recurse infinitely
	var t Tree[int]
	json.Marshal(t)
	xml.Marshal(t)
}

type Tree[T any] struct {
	Node *Node[T]
}

type Node[T any] struct {
	Tree *Tree[T]
}
