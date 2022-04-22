package pkg

import "fmt"

type Node interface { //@ used(true)
	position() int //@ used(true)
}

type noder struct{} //@ used(true)

func (noder) position() int { panic("unreachable") } //@ used(true)

func Fn() { //@ used(true)
	nodes := []Node{struct {
		noder //@ used(true)
	}{}}
	fmt.Println(nodes)
}
