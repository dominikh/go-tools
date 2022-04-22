// Test of field usage detection

package pkg

type t15 struct { //@ used(true)
	f151 int //@ used(true)
}
type a2 [1]t15 //@ used(true)

type t16 struct{} //@ used(true)
type a3 [1][1]t16 //@ used(true)

func foo() { //@ used(true)
	_ = a2{0: {1}}
	_ = a3{{{}}}
}

func init() { foo() } //@ used(true)
