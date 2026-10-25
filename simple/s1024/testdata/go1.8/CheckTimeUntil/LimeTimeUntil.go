package pkg

import "time"

type embedTime struct {
	time.Time
}

func fn(t time.Time) {
	t.Sub(time.Now()) //@ diag(`time.Until`)
	t.Sub(t)
	t2 := time.Now()
	t.Sub(t2)
}

func fnEmbedded(t embedTime) {
	// Sub is promoted from the embedded time.Time; 'time.Until(t)' would not compile.
	t.Sub(time.Now()) //@ diag(`time.Until`)
}

func fnPointer(t *time.Time) {
	// 'time.Until(t)' would not compile either, t needs to be dereferenced.
	t.Sub(time.Now()) //@ diag(`time.Until`)
}
