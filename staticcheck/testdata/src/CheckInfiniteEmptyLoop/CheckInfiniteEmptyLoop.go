package pkg

func fn2() bool { return true }

func fn() {
	for { // MATCH /this loop will spin/
	}

	for fn2() {
	}

	for {
		break
	}

	for true { // MATCH "loop condition never changes"
	}

	x := true
	for x { // MATCH "loop condition never changes"
	}

	x = false
	for x { // MATCH "loop condition never changes"
	}

	for false {
	}

	false := true
	for false { // MATCH "loop condition never changes"
	}
}

// MATCH:16 "this loop will spin"
// MATCH:20 "this loop will spin"
// MATCH:24 "this loop will spin"
// MATCH:31 "this loop will spin"
