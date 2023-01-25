package pkg

type I interface { //@ used(true)
	fn1() //@ used(true)
}

type t struct{} //@ used(true)

func (t) fn1() {} //@ used(true)
func (t) fn2() {} //@ used(false)

func init() { //@ used(true)
	_ = t{}
}

type I1 interface { //@ used(true)
	Foo() //@ used(true)
}

type I2 interface { //@ used(true)
	Foo() //@ used(true)
	bar() //@ used(true)
}

type i3 interface { //@ used(false)
	foo()
	bar()
}

type t1 struct{} //@ used(true)
type t2 struct{} //@ used(true)
type t3 struct{} //@ used(true)
type t4 struct { //@ used(true)
	t3 //@ used(true)
}

func (t1) Foo() {} //@ used(true)
func (t2) Foo() {} //@ used(true)
func (t2) bar() {} //@ used(true)
func (t3) Foo() {} //@ used(true)
func (t3) bar() {} //@ used(true)

func Fn() { //@ used(true)
	var v1 t1
	var v2 t2
	var v3 t3
	var v4 t4
	_ = v1
	_ = v2
	_ = v3
	var x interface{} = v4
	_ = x.(I2)
}
