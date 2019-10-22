package pkg

import "os"

func fn1(x *int) {
	_ = *x // want `possible nil pointer dereference`
	if x != nil {
		return
	}
}

func fn2(x *int) {
	if x == nil {
		println("we should return")
	}
	_ = *x // want `possible nil pointer dereference`
}

func fn3(x *int) {
	if x != nil {
		_ = *x
	}
}

func fn4(x *int) {
	if x == nil {
		x = gen()
	}
	_ = *x
}

func fn5(x *int) {
	if x == nil {
		x = gen()
	}
	_ = *x // want `possible nil pointer dereference`
	if x == nil {
		println("we should return")
	}
}

func fn6() {
	x := new(int)
	if x == nil {
		println("we should return")
	}
	// x can't be nil
	_ = *x
}

func fn7() {
	var x int
	y := &x
	if y == nil {
		println("we should return")
	}
	// y can't be nil
	_ = *y
}

func fn8(x *int) {
	if x == nil {
		return
	}
	// x can't be nil
	_ = *x
}

func fn9(x *int) {
	if x != nil {
		return
	}
	// TODO(dh): not currently supported
	_ = *x
}

func gen() *int { return nil }

func die1(b bool) {
	if b {
		println("yay")
		os.Exit(0)
	} else {
		println("nay")
		os.Exit(1)
	}
}

func die2(b bool) {
	if b {
		println("yay")
		os.Exit(0)
	}
}

func fn10(x *int) {
	if x == nil {
		die1(true)
	}
	_ = *x
}

func fn11(x *int) {
	if x == nil {
		die2(true)
	}
	_ = *x // want `possible nil pointer dereference`
}
