package pkg

import timelib "time"

func fn() {
	t1 := timelib.Now()
	_ = timelib.Now().Sub(t1) //@ diag(`time.Since`)
}
