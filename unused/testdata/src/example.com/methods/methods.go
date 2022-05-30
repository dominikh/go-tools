package pkg

type t1 struct{} //@ used("t1", true)
type t2 struct { //@ used("t2", true)
	t3 //@ used("t3", true)
}
type t3 struct{} //@ used("t3", true)

func (t1) Foo() {} //@ used("Foo", true)
func (t3) Foo() {} //@ used("Foo", true)
func (t3) foo() {} //@ used("foo", false)

func init() { //@ used("init", true)
	_ = t1{}
	_ = t2{}
}
