package pkg

type t1 struct { //@ used(true)
	a int //@ used(true)
	b int //@ used(false)
}

type t2 struct { //@ used(true)
	a int //@ used(false)
	b int //@ used(true)
}

func Fn() { //@ used(true)
	x := t1{}
	y := t2{}
	println(x.a)
	println(y.b)
}
