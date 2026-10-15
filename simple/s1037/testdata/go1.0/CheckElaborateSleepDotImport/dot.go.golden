package pkg

import . "time"

func fn() {
	select { //@ diag(`should use time.Sleep`)
	case <-After(0):
	}
}
