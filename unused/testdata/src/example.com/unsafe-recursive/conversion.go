package pkg

// https://staticcheck.io/issues/1249

import "unsafe"

type t1 struct { //@ used("t1", true)
	f1  int      //@ used("f1", true)
	f2  t2       //@ used("f2", true)
	f3  *t3      //@ used("f3", true)
	t4           //@ used("t4", true)
	*t5          //@ used("t5", true)
	f6  [5]t6    //@ used("f6", true)
	f7  [5][5]t7 //@ used("f7", true)
	f8  nt1      //@ used("f8", true)
	f9  nt2      //@ used("f9", true)
}

type nt1 *t8   //@ used("nt1", true)
type nt2 [4]t9 //@ used("nt2", true)

type t2 struct{ f int } //@ used("t2", true), used("f", true)
type t3 struct{ f int } //@ used("t3", true), used("f", false)
type t4 struct{ f int } //@ used("t4", true), used("f", true)
type t5 struct{ f int } //@ used("t5", true), used("f", false)
type t6 struct{ f int } //@ used("t6", true), used("f", true)
type t7 struct{ f int } //@ used("t7", true), used("f", true)
type t8 struct{ f int } //@ used("t8", true), used("f", false)
type t9 struct{ f int } //@ used("t9", true), used("f", true)

func Foo(x t1) { //@ used("Foo", true), used("x", true)
	_ = unsafe.Pointer(&x)
}
