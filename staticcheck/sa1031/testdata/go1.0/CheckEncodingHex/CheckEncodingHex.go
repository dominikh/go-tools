package pkg

import (
	"encoding/hex"
)

func fn() {
	hex.Encode(nil, nil)
	hex.Encode(make([]byte, 0), nil)
	sliceA := make([]byte, 8)
	sliceB := make([]byte, 8)
	hex.Encode(sliceA, sliceB)
	hex.Encode(sliceA, sliceA) //@ diag(`overlapping dst and src`)
	hex.Encode(sliceA[1:], sliceA[2:])
	hex.Encode(sliceA[1:], sliceA[1:]) //@ diag(`overlapping dst and src`)
	sliceC := sliceA
	hex.Encode(sliceA, sliceC) //@ diag(`overlapping dst and src`)
	if true {
		hex.Encode(sliceA, sliceC) //@ diag(`overlapping dst and src`)
	}
	sliceD := sliceA[1:]
	sliceE := sliceA[1:]
	if true {
		hex.Encode(sliceD, sliceE) //@ diag(`overlapping dst and src`)
	}
	var b bool
	if !b && true {
		hex.Encode(sliceD, sliceE) //@ diag(`overlapping dst and src`)
	}
}

func fooSigmaA(a *[4]byte) {
	low := 2
	x := a[low:]

	if true {
		y := a[low:]
		hex.Encode(x, y) //@ diag(`overlapping dst and src`)
	}
}

func fooSigmaB(a *[4]byte) {
	x := a[:]

	if true {
		y := a[:]
		hex.Encode(x, y) //@ diag(`overlapping dst and src`)
	}
}
