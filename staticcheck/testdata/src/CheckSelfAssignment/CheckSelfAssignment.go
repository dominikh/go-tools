package pkg

func fn(x int) {
	var z int
	var y int
	x = x             // want `self-assignment`
	y = y             // want `self-assignment`
	y, x, z = y, x, 1 // want `self-assignment of y to y` `self-assignment of x to x`
	y = x
	_ = y
	_ = x
	_ = z
	func() {
		x := x
		println(x)
	}()
}

func fn1() {
	var (
		x  []byte
		ch chan int
	)
	x[42] = x[42]                         // want `self-assignment`
	x[pure(42)] = x[pure(42)]             // want `self-assignment`
	x[pure(pure(42))] = x[pure(pure(42))] // want `self-assignment`
	x[impure(42)] = x[impure(42)]
	x[impure(pure(42))] = x[impure(pure(42))]
	x[pure(impure(42))] = x[pure(impure(42))]
	x[pure(<-ch)] = x[pure(<-ch)]
	x[pure(pure(<-ch))] = x[pure(pure(<-ch))]
	x[<-ch] = x[<-ch]
}

func pure(n int) int {
	return n
}

func impure(n int) int {
	println(n)
	return n
}
