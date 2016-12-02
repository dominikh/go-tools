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

func gen() int {
	println() // make it unpure
	return 0
}

func fn2() {
	x, y := gen(), gen()
	x, y = gen(), gen()
	println(x, y)
}

// MATCH:23 /this value of x is never used/
// MATCH:23 /this value of y is never used/

func fn3() {
	x := uint32(0)
	if true {
		x = 1
	} else {
		x = 2
	}
	println(x)
}
