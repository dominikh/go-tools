package pkg

import "math"

func fn(x int) {
	math.Ceil(float64(x))      //@ diag(`on a converted integer is pointless`)
	math.Floor(float64(x * 2)) //@ diag(`on a converted integer is pointless`)
}
