package pkg

func fn1() {
	if true { //@ diag(`empty branch`)
	}
	if true { //@ diag(`empty branch`)
	} else { //@ diag(`empty branch`)
	}
	if true {
		println()
	}

	if true {
		println()
	} else { //@ diag(`empty branch`)
	}

	if true { //@ diag(`empty branch`)
		// TODO handle error
	}

	if true {
	} else {
		println()
	}

	if true {
	} else if false { //@ diag(`empty branch`)
	}
}
