package pkg

type I interface { //@ used("I", true)
	f1() //@ used("f1", true)
	f2() //@ used("f2", true)
}

func init() { //@ used("init", true)
	var _ I //@ used("_", true)
}

type t1 struct{} //@ used("t1", true)
type T2 struct { //@ used("T2", true)
	t1 //@ used("t1", true)
}

func (t1) f1() {} //@ used("f1", true)
func (T2) f2() {} //@ used("f2", true)

func Fn() { //@ used("Fn", true)
	var v T2 //@ used("v", true)
	_ = v.t1
}

type I2 interface { //@ used("I2", true)
	f3() //@ used("f3", true)
	f4() //@ used("f4", true)
}

type t3 struct{} //@ used("t3", true)
type t4 struct { //@ used("t4", true)
	x  int //@ used("x", false)
	y  int //@ used("y", false)
	t3     //@ used("t3", true)
}

func (*t3) f3() {} //@ used("f3", true)
func (*t4) f4() {} //@ used("f4", true)

func init() { //@ used("init", true)
	var i I2 = &t4{} //@ used("i", true)
	i.f3()
	i.f4()
}

type i3 interface { //@ used("i3", true)
	F() //@ used("F", true)
}

type I4 interface { //@ used("I4", true)
	i3
}

type T5 struct { //@ used("T5", true)
	t6 //@ used("t6", true)
}

type t6 struct { //@ used("t6", true)
	F int //@ used("F", true)
}

type t7 struct { //@ used("t7", true)
	X int //@ used("X", true)
}
type t8 struct { //@ used("t8", true)
	t7 //@ used("t7", true)
}
type t9 struct { //@ used("t9", true)
	t8 //@ used("t8", true)
}

var _ = t9{} //@ used("_", true)

type t10 struct{} //@ used("t10", true)

func (*t10) Foo() {} //@ used("Foo", true)

type t11 struct { //@ used("t11", true)
	t10 //@ used("t10", true)
}

var _ = t11{} //@ used("_", true)

type i5 interface{} //@ used("i5", true)
type I6 interface { //@ used("I6", true)
	i5
}

// When recursively looking for embedded exported fields, don't visit top-level type again
type t12 struct { //@ used("t12", true)
	*t12     //@ used("t12", false)
	F    int //@ used("F", true)
}

var _ = t12{} //@ used("_", true)

// embedded fields whose names are exported are used, same as normal exported fields.
type T13 struct { //@ used("T13", true)
	T14  //@ used("T14", true)
	*T15 //@ used("T15", true)
}

type T14 struct{} //@ used("T14", true)
type T15 struct{} //@ used("T15", true)
