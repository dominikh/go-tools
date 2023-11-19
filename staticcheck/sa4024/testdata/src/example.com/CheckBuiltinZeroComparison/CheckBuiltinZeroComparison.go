package pkg

func fn1() {
	var foo []int

	if len(foo) < 0 { //@ diag(`len does not return negative values`)
		println("test")
	}

	switch {
	case len(foo) < 0: //@ diag(`negative`)
		println("test")
	}

	for len(foo) < 0 { //@ diag(`negative`)
		println("test")
	}

	println(len(foo) < 0) //@ diag(`negative`)

	if 0 > cap(foo) { //@ diag(`cap does not return negative values`)
		println("test")
	}

	switch {
	case 0 > cap(foo): //@ diag(`negative`)
		println("test")
	}

	for 0 > cap(foo) { //@ diag(`negative`)
		println("test")
	}

	println(0 > cap(foo)) //@ diag(`negative`)
}

func fn2() {
	const zero = 0
	var foo []int
	println(len(foo) < zero)
	println(len(foo) < 1)
}
