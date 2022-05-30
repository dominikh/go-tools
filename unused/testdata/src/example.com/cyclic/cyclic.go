package pkg

func a() { //@ used("a", false)
	b()
}

func b() { //@ used("b", false)
	a()
}
