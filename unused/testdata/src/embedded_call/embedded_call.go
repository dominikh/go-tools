package pkg

var t1 struct { //@ used(true)
	t2 //@ used(true)
	t3 //@ used(true)
	t4 //@ used(true)
}

type t2 struct{} //@ used(true)
type t3 struct{} //@ used(true)
type t4 struct { //@ used(true)
	t5 //@ used(true)
}
type t5 struct{} //@ used(true)

func (t2) foo() {} //@ used(true)
func (t3) bar() {} //@ used(true)
func (t5) baz() {} //@ used(true)
func init() { //@ used(true)
	t1.foo()
	_ = t1.bar
	t1.baz()
}
