//go:build go1.18

package pkg

type c1 struct{} // used
type c2 struct{} // used
type c3 struct{} // used
type c4 struct{} // used
type c5 struct{} // used
type c6 struct{} // used
type c7 struct{} // used
// c8 should be unused, but see https://staticcheck.io/issues/1199
type c8 struct{} // used
type c9 struct{} // used

type S1[T c1] struct{}  // used
type S2[T any] struct{} // used
type S3 S2[c2]          // used

type I interface { // used
	c3 | c9
}

func Fn1[T c4]()  {} // used
func fn2[T any]() {} // used
func Fn5[T any]() {} // used
func Fn6[T any]() {} // used

var _ = fn2[c5]

func Fn3() { // used
	Fn5[c6]()
	_ = S2[c7]{}
}

func uncalled() { // unused
	_ = Fn6[c8]
}

type S4[T any] struct{} // used

func (S4[T]) usedGenerically()  {} // used
func (S4[T]) usedInstantiated() {} // used
func (recv S4[T]) Exported() { // used
	recv.usedGenerically()
}
func (S4[T]) unused() {} // unused

func Fn4() { // used
	var x S4[int]
	x.usedInstantiated()
}

type s1[T any] struct{} // unused

func (recv s1[a]) foo() { recv.foo(); recv.bar(); recv.baz() } // unused
func (recv s1[b]) bar() { recv.foo(); recv.bar(); recv.baz() } // unused
func (recv s1[c]) baz() { recv.foo(); recv.bar(); recv.baz() } // unused

func fn7[T interface{ foo() }]() {} // unused
func fn8[T struct { // unused
	x int
}]() {
}
func Fn9[T struct { // used
	X *s2 // used
}]() {
}

type s2 struct{} // used

func fn10[E any](x []E) {} // unused

type Tree[T any] struct { // used
	Root *Node[T] // used
}

type Node[T any] struct { // used
	Tree *Tree[T] // used
}

type foo struct{} // used

type Bar *Node[foo] // used

func (n Node[T]) anyMethod() {} // unused

func fn11[T ~struct{ Field int }]() { // unused
	// don't crash because of the composite literal
	_ = T{Field: 42}
}

type convertGeneric1 struct { // used
	field int // used
}

type convertGeneric2 struct { // used
	field int // used
}

var _ = convertGeneric1{}.field // mark field as used

func Fn12[T1 convertGeneric1, T2 convertGeneric2](a T1) { // used
	_ = T2(a) // conversion marks T2.field as used
}
