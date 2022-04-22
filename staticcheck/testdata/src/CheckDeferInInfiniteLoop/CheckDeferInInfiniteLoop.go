package pkg

func fn() {
	for {
		defer println() //@ diag(`will never run`)
	}
	for {
		defer println() //@ diag(`will never run`)
		go func() {
			return
		}()
	}
	for {
		defer println()
		return
	}
	for false {
		defer println()
	}
	for {
		defer println()
		break
	}
	for {
		defer println()
		if true {
			break
		}
	}
	for {
		defer println()
		// Right now, we treat every break the same. We could analyse
		// this further and see, that the break doesn't break out of
		// the outer loop.
		for {
			break
		}
	}
}
