package pkg

func fn2() bool { return true }

func fn() {
	for { //@ diag(`this loop will spin`)
	}

	for fn2() {
	}

	for {
		break
	}

	for true { //@ diag(`loop condition never changes`), diag(`this loop will spin`)
	}

	x := true
	for x { //@ diag(`loop condition never changes`), diag(`this loop will spin`)
	}

	x = false
	for x { //@ diag(`loop condition never changes`), diag(`this loop will spin`)
	}

	for false {
	}

	false := true
	for false { //@ diag(`loop condition never changes`), diag(`this loop will spin`)
	}
}
