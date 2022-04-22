package pkg

func fn1(x interface{}) {
	if x, ok := x.(int); ok {
		_ = x
	} else {
		_ = x //@ diag(`x refers to the result of a failed type assertion`)
		x = 1
		// No diagnostic, x is no longer the zero value
		_ = x
	}

	if x, ok := x.(int); ok {
	} else {
		// No diagnostic because x escapes
		_ = x
		_ = &x
	}

	if y, ok := x.(int); ok {
		_ = y
	} else {
		// No diagnostic because x isn't shadowed
		_ = x
	}

	if y, ok := x.(int); ok {
		_ = y
	} else {
		// No diagnostic because y isn't shadowing x
		_ = y
	}

	if x, ok := x.(int); ok && true {
		_ = x
	} else {
		// No diagnostic because the condition isn't simply 'ok'
		_ = x
	}

	if x, ok := x.(*int); ok {
		_ = x
	} else if x != nil { //@ diag(`x refers to`)
	} else if x == nil { //@ diag(`x refers to`)
	}
}
