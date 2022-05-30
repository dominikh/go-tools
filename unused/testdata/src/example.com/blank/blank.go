package pkg

import _ "fmt"

type t1 struct{} //@ used("t1", false)
type t2 struct { //@ used("t2", true)
	_ int //@ used("_", true)
}
type t3 struct{} //@ used("t3", true)
type t4 struct{} //@ used("t4", true)
type t5 struct{} //@ used("t5", true)

var _ = t2{} //@ used("_", true)

func fn1() { //@ used("fn1", false)
	_ = t1{}
	var _ = t1{} //@ quiet("_")
}

func fn2() { //@ used("fn2", true)
	_ = t3{}
	var _ t4        //@ used("_", true)
	var _ *t5 = nil //@ used("_", true)
}

func init() { //@ used("init", true)
	fn2()
}

func _() {} //@ used("_", true)

type _ struct{} //@ used("_", true)
