package pkg

import t "time"

func fn(x struct{ After func() t.Duration }) {
	select { //@ diag(`should use time.Sleep`)
	case <-t.After(x.After()):
	}
}
