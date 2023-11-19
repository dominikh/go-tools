package pkg

import (
	"encoding/ascii85"
)

func fn() {
	ascii85.Encode(nil, nil)
	ascii85.Encode(make([]byte, 0), nil)
	sliceA := make([]byte, 8)
	sliceB := make([]byte, 8)
	ascii85.Encode(sliceA, sliceB)
	ascii85.Encode(sliceA, sliceA) //@ diag(`overlapping dst and src`)
	ascii85.Encode(sliceA[1:], sliceA[2:])
	ascii85.Encode(sliceA[1:], sliceA[1:]) //@ diag(`overlapping dst and src`)
	sliceC := sliceA
	ascii85.Encode(sliceA, sliceC) //@ diag(`overlapping dst and src`)
	if true {
		ascii85.Encode(sliceA, sliceC) //@ diag(`overlapping dst and src`)
	}
	sliceD := sliceA[1:]
	sliceE := sliceA[1:]
	if true {
		ascii85.Encode(sliceD, sliceE) //@ diag(`overlapping dst and src`)
	}
}

func fooSigmaA(a *[4]byte) {
	low := 2
	x := a[low:]

	if true {
		y := a[low:]
		ascii85.Encode(x, y) //@ diag(`overlapping dst and src`)
	}
}

func fooSigmaB(a *[4]byte) {
	x := a[:]

	if true {
		y := a[:]
		ascii85.Encode(x, y) //@ diag(`overlapping dst and src`)
	}
}
