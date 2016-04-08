package pkg

func fn2() int { return 0 }

func fn() {
	for { // MATCH /infinite empty loop/
	}

	for fn2() {
	}

	for {
		break
	}
}
