package pkg

var a byte     //@ used("a", true)
var b [16]byte //@ used("b", true)

type t1 struct{} //@ used("t1", true)
type t2 struct{} //@ used("t2", true)
type t3 struct{} //@ used("t3", true)
type t4 struct{} //@ used("t4", true)
type t5 struct{} //@ used("t5", true)

type iface interface{} //@ used("iface", true)

var x t1           //@ used("x", true)
var y = t2{}       //@ used("y", true)
var j = t3{}       //@ used("j", true)
var k = t4{}       //@ used("k", true)
var l iface = t5{} //@ used("l", true)

func Fn() { //@ used("Fn", true)
	println(a)
	_ = b[:]

	_ = x
	_ = y
	_ = j
	_ = k
	_ = l
}
