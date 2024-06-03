package pkg

import "time"

func fn1() {
	// Not flagged because this is no longer a problem in Go 1.23.
	for range time.Tick(0) {
		println("")
		if true {
			break
		}
	}
}
