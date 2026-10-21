package pkg

func fn() {
	var a, b, c bool
	var e, f, g int
	var h, i float64

	_ = !(a && b && (!c || e > f) && g == f) //@ diag(`could apply De Morgan's laws`)
	_ = !(a && h > i)
}
