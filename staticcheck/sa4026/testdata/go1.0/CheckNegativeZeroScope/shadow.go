package pkg

import m "math"

var _ = m.Copysign

func shadow() {
	m := 1
	_ = m
	_ = -0.0 //@ diag(`does not produce a negative zero`)
	_ = float32(-0) //@ diag(`does not produce a negative zero`)
}
