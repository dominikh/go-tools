//go:build go1.18

package pkg

type c1 struct{} //@ used(true)
type c2 struct{} //@ used(true)
type c3 struct{} //@ used(true)
type c4 struct{} //@ used(true)
type c5 struct{} //@ used(true)
type c6 struct{} //@ used(true)
type c7 struct{} //@ used(true)
// c8 should be unused, but see https://staticcheck.io/issues/1199
type c8 struct{} //@ used(true)
type c9 struct{} //@ used(true)

type S1[T c1] struct{}  //@ used(true)
type S2[T any] struct{} //@ used(true)
type S3 S2[c2]          //@ used(true)

type I interface { //@ used(true)
	c3 | c9
}

func Fn1[T c4]()  {} //@ used(true)
func fn2[T any]() {} //@ used(true)
func Fn5[T any]() {} //@ used(true)
func Fn6[T any]() {} //@ used(true)

var _ = fn2[c5]

func Fn3() { //@ used(true)
	Fn5[c6]()
	_ = S2[c7]{}
}

func uncalled() { //@ used(false)
	_ = Fn6[c8]
}

type S4[T any] struct{} //@ used(true)

func (S4[T]) usedGenerically()  {} //@ used(true)
func (S4[T]) usedInstantiated() {} //@ used(true)
func (recv S4[T]) Exported() { //@ used(true)
	recv.usedGenerically()
}
func (S4[T]) unused() {} //@ used(false)

func Fn4() { //@ used(true)
	var x S4[int]
	x.usedInstantiated()
}

type s1[T any] struct{} //@ used(false)

func (recv s1[a]) foo() { recv.foo(); recv.bar(); recv.baz() } //@ used(false)
func (recv s1[b]) bar() { recv.foo(); recv.bar(); recv.baz() } //@ used(false)
func (recv s1[c]) baz() { recv.foo(); recv.bar(); recv.baz() } //@ used(false)

func fn7[T interface{ foo() }]() {} //@ used(false)
func fn8[T struct { //@ used(false)
	x int
}]() {
}
func Fn9[T struct { //@ used(true)
	X *s2 //@ used(true)
}]() {
}

type s2 struct{} //@ used(true)

func fn10[E any](x []E) {} //@ used(false)

type Tree[T any] struct { //@ used(true)
	Root *Node[T] //@ used(true)
}

type Node[T any] struct { //@ used(true)
	Tree *Tree[T] //@ used(true)
}

type foo struct{} //@ used(true)

type Bar *Node[foo] //@ used(true)

func (n Node[T]) anyMethod() {} //@ used(false)

func fn11[T ~struct{ Field int }]() { //@ used(false)
	// don't crash because of the composite literal
	_ = T{Field: 42}
}

type convertGeneric1 struct { //@ used(true)
	field int //@ used(true)
}

type convertGeneric2 struct { //@ used(true)
	field int //@ used(true)
}

var _ = convertGeneric1{}.field // mark field as used

func Fn12[T1 convertGeneric1, T2 convertGeneric2](a T1) { //@ used(true)
	_ = T2(a) // conversion marks T2.field as used
}
