package pkg

func a() { // MATCH a
	b()
}

func b() { // MATCH b
	a()
}
