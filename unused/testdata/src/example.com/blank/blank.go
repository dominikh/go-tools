package pkg

import _ "fmt"

type t1 struct{} //@ used(false)
type t2 struct { //@ used(true)
	_ int //@ used(true)
}
type t3 struct{} //@ used(true)
type t4 struct{} //@ used(true)
type t5 struct{} //@ used(true)

var _ = t2{}

func fn1() { //@ used(false)
	_ = t1{}
	var _ = t1{}
}

func fn2() { //@ used(true)
	_ = t3{}
	var _ t4
	var _ *t5 = nil
}

func init() { //@ used(true)
	fn2()
}

func _() {}

type _ struct{}
