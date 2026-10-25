package pkg

import timelib "time"

func fn(t timelib.Time) {
	t.Sub(timelib.Now()) //@ diag(`time.Until`)
}
