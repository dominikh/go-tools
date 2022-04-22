package pkg

func fn() {
	var a int
	var b uint
	_ = a % 1 //@ diag(`x % 1 is always zero`)
	_ = a % 2
	_ = b % 1 //@ diag(`x % 1 is always zero`)
}
