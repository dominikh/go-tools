// Test of field usage detection

package pkg

type t15 struct { //@ used("t15", true)
	f151 int //@ used("f151", true)
}
type a2 [1]t15 //@ used("a2", true)

type t16 struct{} //@ used("t16", true)
type a3 [1][1]t16 //@ used("a3", true)

func foo() { //@ used("foo", true)
	_ = a2{0: {1}}
	_ = a3{{{}}}
}

func init() { foo() } //@ used("init", true)
