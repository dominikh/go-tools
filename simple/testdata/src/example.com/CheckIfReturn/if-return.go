package pkg

func fn() bool { return true }
func fn1() bool {
	x := true
	if x { //@ diag(`should use 'return x'`)
		return true
	}
	return false
}

func fn2() bool {
	x := true
	if !x {
		return true
	}
	if x {
		return true
	}
	return false
}

func fn3() int {
	var x bool
	if x {
		return 1
	}
	return 2
}

func fn4() bool { return true }

func fn5() bool {
	if fn() { //@ diag(`should use 'return !fn()'`)
		return false
	}
	return true
}

func fn6() bool {
	if fn3() != fn3() { //@ diag(`should use 'return fn3() != fn3()'`)
		return true
	}
	return false
}

func fn7() bool {
	if 1 > 2 { //@ diag(`should use 'return 1 > 2'`)
		return true
	}
	return false
}

func fn8() bool {
	if fn() || fn() {
		return true
	}
	return false
}

func fn9(x int) bool {
	if x > 0 {
		return true
	}
	return true
}

func fn10(x int) bool {
	if x > 0 { //@ diag(`should use 'return x <= 0'`)
		return false
	}
	return true
}

func fn11(x bool) bool {
	if x { //@ diag(`should use 'return !x'`)
		return false
	}
	return true
}

func fn12() bool {
	var x []bool
	if x[0] { //@ diag(`should use 'return !x[0]'`)
		return false
	}
	return true
}

func fn13(a, b int) bool {
	if a != b { //@ diag(`should use 'return a == b' instead of 'if a != b`)
		return false
	}
	return true
}

func fn14(a, b int) bool {
	if a >= b { //@ diag(`should use 'return a < b' instead of 'if a >= b`)
		return false
	}
	return true
}

func fn15() bool {
	if !fn() { //@ diag(`should use 'return fn()'`)
		return false
	}
	return true
}

func fn16() <-chan bool {
	x := make(chan bool, 1)
	x <- true
	return x
}

func fn17() bool {
	if <-fn16() { //@ diag(`should use 'return !<-fn16()'`)
		return false
	}
	return true
}

func fn18() *bool {
	x := true
	return &x
}

func fn19() bool {
	if *fn18() { //@ diag(`should use 'return !*fn18()'`)
		return false
	}
	return true
}

const a = true
const b = false

func fn20(x bool) bool {
	// Don't match on constants other than the predeclared true and false. This protects us both from build tag woes,
	// and from code that breaks when the constant values change.
	if x {
		return a
	}
	return b
}

func fn21(x bool) bool {
	// Don't flag, 'true' isn't the predeclared identifier.
	const true = false
	if x {
		return true
	}
	return false
}
