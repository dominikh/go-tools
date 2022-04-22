package baz

import "fmt"

type Foo interface { //@ used(true)
	bar() //@ used(true)
}

func Bar(f Foo) { //@ used(true)
	f.bar()
}

type Buzz struct{} //@ used(true)

func (b *Buzz) bar() { //@ used(true)
	fmt.Println("foo bar buzz")
}
