package pkg

func fn() {
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
