package pkg

type I interface { //@ used(true)
	f1() //@ used(true)
	f2() //@ used(true)
}

func init() { //@ used(true)
	var _ I
}

type t1 struct{} //@ used(true)
type T2 struct { //@ used(true)
	t1 //@ used(true)
}

func (t1) f1() {} //@ used(true)
func (T2) f2() {} //@ used(true)

func Fn() { //@ used(true)
	var v T2
	_ = v.t1
}

type I2 interface { //@ used(true)
	f3() //@ used(true)
	f4() //@ used(true)
}

type t3 struct{} //@ used(true)
type t4 struct { //@ used(true)
	x  int //@ used(false)
	y  int //@ used(false)
	t3     //@ used(true)
}

func (*t3) f3() {} //@ used(true)
func (*t4) f4() {} //@ used(true)

func init() { //@ used(true)
	var i I2 = &t4{}
	i.f3()
	i.f4()
}

type i3 interface { //@ used(true)
	F() //@ used(true)
}

type I4 interface { //@ used(true)
	i3
}

type T5 struct { //@ used(true)
	t6 //@ used(true)
}

type t6 struct { //@ used(true)
	F int //@ used(true)
}

type t7 struct { //@ used(true)
	X int //@ used(true)
}
type t8 struct { //@ used(true)
	t7 //@ used(true)
}
type t9 struct { //@ used(true)
	t8 //@ used(true)
}

var _ = t9{}

type t10 struct{} //@ used(true)

func (*t10) Foo() {} //@ used(true)

type t11 struct { //@ used(true)
	t10 //@ used(true)
}

var _ = t11{}

type i5 interface{} //@ used(true)
type I6 interface { //@ used(true)
	i5
}
