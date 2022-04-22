package pkg

func fn() {
	var ch chan int
	for {
		switch {
		case true:
			break //@ diag(`ineffective break statement`)
		default:
			break //@ diag(`ineffective break statement`)
		}
	}

	for {
		select {
		case <-ch:
			break //@ diag(`ineffective break statement`)
		}
	}

	for {
		switch {
		case true:
		}

		switch {
		case true:
			break //@ diag(`ineffective break statement`)
		}

		switch {
		case true:
		}
	}

	for {
		switch {
		case true:
			if true {
				break //@ diag(`ineffective break statement`)
			} else {
				break //@ diag(`ineffective break statement`)
			}
		}
	}

	for {
		switch {
		case true:
			if true {
				break
			}

			println("do work")
		}
	}

label:
	for {
		switch {
		case true:
			break label
		}
	}

	for range ([]int)(nil) {
		switch {
		default:
			break //@ diag(`ineffective break statement`)
		}
	}
}
