package pkg

import timelib "time"

func fn() {
	select { //@ diag(`should use time.Sleep`)
	case <-timelib.After(0):
	}
}
