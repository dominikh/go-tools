package pkg

import . "time"

func fn(t Time) {
	_ = Now().Sub(t) //@ diag(`time.Since`)
}
