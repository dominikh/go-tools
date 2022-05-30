package baz

import "fmt"

type Foo interface { //@ used("Foo", true)
	bar() //@ used("bar", true)
}

func Bar(f Foo) { //@ used("Bar", true), used("f", true)
	f.bar()
}

type Buzz struct{} //@ used("Buzz", true)

func (b *Buzz) bar() { //@ used("bar", true), used("b", true)
	fmt.Println("foo bar buzz")
}
