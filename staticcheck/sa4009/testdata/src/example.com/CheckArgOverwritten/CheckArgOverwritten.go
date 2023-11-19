package pkg

var x = func(arg int) { //@ diag(`overwritten`)
	arg = 1
	println(arg)
}
