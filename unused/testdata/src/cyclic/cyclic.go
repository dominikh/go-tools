package pkg

func a() { //@ used(false)
	b()
}

func b() { //@ used(false)
	a()
}
