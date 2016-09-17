package pkg

type t1 struct {
	a int
	b int
}

type t2 struct {
	a int
	b int
}

type t3 t1

func fn() {
	v1 := t1{1, 2}
	_ = t2{v1.a, v1.b}       // MATCH /should use type conversion/
	_ = t2{a: v1.a, b: v1.b} // MATCH /should use type conversion/
	_ = t2{b: v1.b, a: v1.a} // MATCH /should use type conversion/
	_ = t3{v1.a, v1.b}       // MATCH /should use type conversion/

	_ = t2{v1.b, v1.a}
	_ = t2{a: v1.b, b: v1.a}
	_ = t2{a: v1.a}
	_ = t1{v1.a, v1.b}
}
