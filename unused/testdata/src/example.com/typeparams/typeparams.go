//go:build go1.18

package pkg

type c1 struct{} //@ used("c1", true)
type c2 struct{} //@ used("c2", true)
type c3 struct{} //@ used("c3", true)
type c4 struct{} //@ used("c4", true)
type c5 struct{} //@ used("c5", true)
type c6 struct{} //@ used("c6", true)
type c7 struct{} //@ used("c7", true)
type c8 struct{} //@ used("c8", false)
type c9 struct{} //@ used("c9", true)

type S1[T c1] struct{}  //@ used("S1", true), used("T", true)
type S2[T any] struct{} //@ used("S2", true), used("T", true)
type S3 S2[c2]          //@ used("S3", true)

type I interface { //@ used("I", true)
	c3 | c9
}

func Fn1[T c4]()  {} //@ used("Fn1", true), used("T", true)
func fn2[T any]() {} //@ used("fn2", true), used("T", true)
func Fn5[T any]() {} //@ used("Fn5", true), used("T", true)
func Fn6[T any]() {} //@ used("Fn6", true), used("T", true)

var _ = fn2[c5] //@ used("_", true)

func Fn3() { //@ used("Fn3", true)
	Fn5[c6]()
	_ = S2[c7]{}
}

func uncalled() { //@ used("uncalled", false)
	_ = Fn6[c8]
}

type S4[T any] struct{} //@ used("S4", true), used("T", true)

func (S4[T]) usedGenerically()  {} //@ used("usedGenerically", true), used("T", true)
func (S4[T]) usedInstantiated() {} //@ used("usedInstantiated", true), used("T", true)
func (recv S4[T]) Exported() { //@ used("Exported", true), used("recv", true), used("T", true)
	recv.usedGenerically()
}
func (S4[T]) unused() {} //@ used("unused", false), quiet("T")

func Fn4() { //@ used("Fn4", true)
	var x S4[int] //@ used("x", true)
	x.usedInstantiated()
}

type s1[T any] struct{} //@ used("s1", false), quiet("T")

func (recv s1[a]) foo() { recv.foo(); recv.bar(); recv.baz() } //@ used("foo", false), quiet("recv"), quiet("a")
func (recv s1[b]) bar() { recv.foo(); recv.bar(); recv.baz() } //@ used("bar", false), quiet("recv"), quiet("b")
func (recv s1[c]) baz() { recv.foo(); recv.bar(); recv.baz() } //@ used("baz", false), quiet("recv"), quiet("c")

func fn7[T interface { //@ used("fn7", false), quiet("T")
	foo() //@ quiet("foo")
}]() {
}

func fn8[T struct { //@ used("fn8", false), quiet("T")
	x int //@ quiet("x")
}]() {
}

func Fn9[T struct { //@ used("Fn9", true), used("T", true)
	X *s2 //@ used("X", true)
}]() {
}

type s2 struct{} //@ used("s2", true)

func fn10[E any](x []E) {} //@ used("fn10", false), quiet("E"), quiet("x")

type Tree[T any] struct { //@ used("Tree", true), used("T", true)
	Root *Node[T] //@ used("Root", true)
}

type Node[T any] struct { //@ used("Node", true), used("T", true)
	Tree *Tree[T] //@ used("Tree", true)
}

type foo struct{} //@ used("foo", true)

type Bar *Node[foo] //@ used("Bar", true)

func (n Node[T]) anyMethod() {} //@ used("anyMethod", false), quiet("n"), quiet("T")

func fn11[T ~struct { //@ used("fn11", false), quiet("T")
	Field int //@ quiet("Field")
}]() {
	// don't crash because of the composite literal
	_ = T{Field: 42}
}

type convertGeneric1 struct { //@ used("convertGeneric1", true)
	field int //@ used("field", true)
}

type convertGeneric2 struct { //@ used("convertGeneric2", true)
	field int //@ used("field", true)
}

// mark field as used
var _ = convertGeneric1{}.field //@ used("_", true)

func Fn12[T1 convertGeneric1, T2 convertGeneric2](a T1) { //@ used("Fn12", true), used("T1", true), used("T2", true), used("a", true)
	_ = T2(a) // conversion marks T2.field as used
}

type S5[A, B any] struct{} //@ used("S5", true), used("A", true), used("B", true)
type S6 S5[int, string]    //@ used("S6", true)
