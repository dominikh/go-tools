package pkg

import "math"

func fn3[S int8 | int16](x S) {
	math.Ceil(float64(x)) //@ diag(`on a converted integer is pointless`)
}

func fn4[S int8 | int16 | float32](x S) {
	math.Ceil(float64(x))
}
