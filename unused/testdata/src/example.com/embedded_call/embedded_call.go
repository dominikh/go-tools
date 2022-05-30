package pkg

var t1 struct { //@ used("t1", true)
	t2 //@ used("t2", true)
	t3 //@ used("t3", true)
	t4 //@ used("t4", true)
}

type t2 struct{} //@ used("t2", true)
type t3 struct{} //@ used("t3", true)
type t4 struct { //@ used("t4", true)
	t5 //@ used("t5", true)
}
type t5 struct{} //@ used("t5", true)

func (t2) foo() {} //@ used("foo", true)
func (t3) bar() {} //@ used("bar", true)
func (t5) baz() {} //@ used("baz", true)
func init() { //@ used("init", true)
	t1.foo()
	_ = t1.bar
	t1.baz()
}
