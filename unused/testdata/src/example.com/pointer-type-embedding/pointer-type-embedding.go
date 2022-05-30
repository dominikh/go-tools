package pkg

func init() { //@ used("init", true)
	var p P //@ used("p", true)
	_ = p.n
}

type T0 struct { //@ used("T0", true)
	m int //@ used("m", false)
	n int //@ used("n", true)
}

type T1 struct { //@ used("T1", true)
	T0 //@ used("T0", true)
}

type P *T1 //@ used("P", true)
