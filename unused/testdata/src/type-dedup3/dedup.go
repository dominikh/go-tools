package pkg

func fn1(t struct { //@ used(true)
	a int //@ used(true)
	b int //@ used(true)
}) {
	fn2(t)
}

func fn2(t struct { //@ used(true)
	a int //@ used(true)
	b int //@ used(true)
}) {
	println(t.a)
	println(t.b)
}

func Fn() { //@ used(true)
	fn1(struct {
		a int //@ used(true)
		b int //@ used(true)
	}{1, 2})
}
