package pkg

func fn() {
	var x, y int
	var z []int
	var a bool

	if x == 1 || x == 2 { //@ diag(`could use tagged switch on x`)
	} else if x == 3 {
	}

	if x == 1 || x == 2 { //@ diag(`could use tagged switch on x`)
	} else if x == 3 {
	} else {
	}

	if x == 1 || x == 2 {
	} else if y == 3 {
	} else {
	}

	if a == (x == y) { //@ diag(`could use tagged switch on a`)
	} else if a == (x != y) {
	}

	if z[0] == 1 || z[0] == 2 { //@ diag(`could use tagged switch on z[0]`)
	} else if z[0] == 3 {
	}

	for {
		if x == 1 || x == 2 { //@ diag(`could use tagged switch on x`)
		} else if x == 3 {
		}
	}

	for {
		if x == 1 || x == 2 {
		} else if x == 3 {
			break
		}
	}

	if x == 1 || x == 2 {
	}
}

func nilCommonOperand(err, firstErr error) {
	if nil == err {
	} else if nil == firstErr {
	}
}

func parenthesizedNilCommonOperand(err, firstErr, otherErr error) {
	if (nil) == err || (nil) == firstErr {
	} else if (nil) == otherErr {
	}
}

func nilCase(err, firstErr error) {
	if err == nil { //@ diag(`could use tagged switch on err`)
	} else if err == firstErr {
	}
}

func typedNilTag(x, y *int) {
	var zero *int
	if zero == x { //@ diag(`could use tagged switch on zero`)
	} else if zero == y {
	}
}

func typedNilConversion(x, y *int) {
	if (*int)(nil) == x {
	} else if (*int)(nil) == y {
	}
}

func shadowedNilTag(nil, x, y int) {
	if nil == x { //@ diag(`could use tagged switch on nil`)
	} else if nil == y {
	}
}
