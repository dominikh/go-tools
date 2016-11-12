package pkg

func fn1() {
	var x int
	x = gen() // MATCH /this value of x is never used/
	x = gen()
	println(x)

	var y int
	if true {
		y = gen() // MATCH /this value of y is never used/
	}
	y = gen()
	println(y)
}

func gen() int { return 0 }

func fn2() {
	x, y := gen(), gen()
	x, y = gen(), gen()
	println(x, y)
}

// MATCH:20 /this value of x is never used/
// MATCH:20 /this value of y is never used/
