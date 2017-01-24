package pkg

func fn2() bool { return true }

func fn() {
	for { // MATCH /infinite empty loop/
	}

	for fn2() {
	}

	for {
		break
	}
}
