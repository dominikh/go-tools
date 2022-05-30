package pkg

type t1 struct { //@ used("t1", true)
	a int //@ used("a", true)
	b int //@ used("b", false)
}

type t2 struct { //@ used("t2", true)
	a int //@ used("a", false)
	b int //@ used("b", true)
}

func Fn() { //@ used("Fn", true)
	x := t1{} //@ used("x", true)
	y := t2{} //@ used("y", true)
	println(x.a)
	println(y.b)
}
