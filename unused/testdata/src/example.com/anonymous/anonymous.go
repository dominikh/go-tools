package pkg

import "fmt"

type Node interface { //@ used("Node", true)
	position() int //@ used("position", true)
}

type noder struct{} //@ used("noder", true)

func (noder) position() int { panic("unreachable") } //@ used("position", true)

func Fn() { //@ used("Fn", true)
	nodes := []Node{struct { //@ used("nodes", true)
		noder //@ used("noder", true)
	}{}}
	fmt.Println(nodes)
}
