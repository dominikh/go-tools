package pkg

var a byte     //@ used(true)
var b [16]byte //@ used(true)

type t1 struct{} //@ used(true)
type t2 struct{} //@ used(true)
type t3 struct{} //@ used(true)
type t4 struct{} //@ used(true)
type t5 struct{} //@ used(true)

type iface interface{} //@ used(true)

var x t1           //@ used(true)
var y = t2{}       //@ used(true)
var j = t3{}       //@ used(true)
var k = t4{}       //@ used(true)
var l iface = t5{} //@ used(true)

func Fn() { //@ used(true)
	println(a)
	_ = b[:]

	_ = x
	_ = y
	_ = j
	_ = k
	_ = l
}
