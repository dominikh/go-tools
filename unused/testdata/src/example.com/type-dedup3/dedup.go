package pkg

func fn1(t struct { //@ used("fn1", true), used("t", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}) {
	fn2(t)
}

func fn2(t struct { //@ used("fn2", true), used("t", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
}) {
	println(t.a)
	println(t.b)
}

func Fn() { //@ used("Fn", true)
	fn1(struct {
		a int //@ used("a", true)
		b int //@ used("b", true)
	}{1, 2})
}
