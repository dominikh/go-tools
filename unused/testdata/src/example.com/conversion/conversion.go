package pkg

import (
	"compress/flate"
	"unsafe"
)

type t1 struct { //@ used("t1", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}

type t2 struct { //@ used("t2", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}

type t3 struct { //@ used("t3", true)
	a int //@ used("a", true)
	b int //@ used("b", false)
}

type t4 struct { //@ used("t4", true)
	a int //@ used("a", true)
	b int //@ used("b", false)
}

type t5 struct { //@ used("t5", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}

type t6 struct { //@ used("t6", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}

type t7 struct { //@ used("t7", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}

type t8 struct { //@ used("t8", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}

type t9 struct { //@ used("t9", true)
	Offset int64 //@ used("Offset", true)
	Err    error //@ used("Err", true)
}

type t10 struct { //@ used("t10", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}

func fn() { //@ used("fn", true)
	// All fields in t2 used because they're initialised in t1
	v1 := t1{0, 1} //@ used("v1", true)
	v2 := t2(v1)   //@ used("v2", true)
	_ = v2

	// Field b isn't used by anyone
	v3 := t3{}   //@ used("v3", true)
	v4 := t4(v3) //@ used("v4", true)
	println(v3.a)
	_ = v4

	// Both fields are used
	v5 := t5{}   //@ used("v5", true)
	v6 := t6(v5) //@ used("v6", true)
	println(v5.a)
	println(v6.b)

	v7 := &t7{} //@ used("v7", true)
	println(v7.a)
	println(v7.b)
	v8 := (*t8)(v7) //@ used("v8", true)
	_ = v8

	vb := flate.ReadError{} //@ used("vb", true)
	v9 := t9(vb)            //@ used("v9", true)
	_ = v9

	// All fields are used because this is an unsafe conversion
	var b []byte                         //@ used("b", true)
	v10 := (*t10)(unsafe.Pointer(&b[0])) //@ used("v10", true)
	_ = v10
}

func init() { fn() } //@ used("init", true)
