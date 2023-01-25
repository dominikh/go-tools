package pkg

func init() { //@ used(true)
	var p P
	_ = p.n
}

type T0 struct { //@ used(true)
	m int //@ used(false)
	n int //@ used(true)
}

type T1 struct { //@ used(true)
	T0 //@ used(true)
}

type P *T1 //@ used(true)
