package pkg

type I interface { //@ used("I", true)
	fn1() //@ used("fn1", true)
}

type t struct{} //@ used("t", true)

func (t) fn1() {} //@ used("fn1", true)
func (t) fn2() {} //@ used("fn2", false)

func init() { //@ used("init", true)
	_ = t{}
}

type I1 interface { //@ used("I1", true)
	Foo() //@ used("Foo", true)
}

type I2 interface { //@ used("I2", true)
	Foo() //@ used("Foo", true)
	bar() //@ used("bar", true)
}

type i3 interface { //@ used("i3", false)
	foo() //@ quiet("foo")
	bar() //@ quiet("bar")
}

type t1 struct{} //@ used("t1", true)
type t2 struct{} //@ used("t2", true)
type t3 struct{} //@ used("t3", true)
type t4 struct { //@ used("t4", true)
	t3 //@ used("t3", true)
}

func (t1) Foo() {} //@ used("Foo", true)
func (t2) Foo() {} //@ used("Foo", true)
func (t2) bar() {} //@ used("bar", true)
func (t3) Foo() {} //@ used("Foo", true)
func (t3) bar() {} //@ used("bar", true)

func Fn() { //@ used("Fn", true)
	var v1 t1 //@ used("v1", true)
	var v2 t2 //@ used("v2", true)
	var v3 t3 //@ used("v3", true)
	var v4 t4 //@ used("v4", true)
	_ = v1
	_ = v2
	_ = v3
	var x interface{} = v4 //@ used("x", true)
	_ = x.(I2)
}

// Text pointer receivers
type T2 struct{} //@ used("T2", true)

func (*T2) fn1() {} //@ used("fn1", true)
func (*T2) fn2() {} //@ used("fn2", false)
