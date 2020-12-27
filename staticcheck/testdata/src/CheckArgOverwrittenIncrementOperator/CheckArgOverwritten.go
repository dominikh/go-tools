package pkg

var x = func(arg int) { // want `overwritten`
	arg++
	println(arg)
}
