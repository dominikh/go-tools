package pkg

type t1 struct{} //@ used(true)
type t2 struct { //@ used(true)
	t3 //@ used(true)
}
type t3 struct{} //@ used(true)

func (t1) Foo() {} //@ used(true)
func (t3) Foo() {} //@ used(true)
func (t3) foo() {} //@ used(false)

func init() { //@ used(true)
	_ = t1{}
	_ = t2{}
}
