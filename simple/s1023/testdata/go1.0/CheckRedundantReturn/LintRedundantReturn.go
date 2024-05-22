package pkg

func fn1() {
	return //@ diag(`redundant return`)
}

func fn2(a int) {
	return //@ diag(`redundant return`)
}

func fn3() int {
	return 3
}

func fn4() (n int) {
	return
}

func fn5(b bool) {
	if b {
		return
	}
}

func fn6() {
	return
	println("foo")
}

func fn7() {
	return
	println("foo")
	return //@ diag(`redundant return`)
}

func fn8() {
	_ = func() {
		return //@ diag(`redundant return`)
	}
}
