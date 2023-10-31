package pkg

import (
	"encoding/base32"
)

func fn() {
	encoding := base32.StdEncoding
	encoding.Encode(nil, nil)
	encoding.Encode(make([]byte, 0), nil)
	sliceA := make([]byte, 8)
	sliceB := make([]byte, 8)
	encoding.Encode(sliceA, sliceB)
	encoding.Encode(sliceA, sliceA) //@ diag(`overlapping dst and src`)
	encoding.Encode(sliceA[1:], sliceA[2:])
	encoding.Encode(sliceA[1:], sliceA[1:]) //@ diag(`overlapping dst and src`)
	sliceC := sliceA
	encoding.Encode(sliceA, sliceC) //@ diag(`overlapping dst and src`)
	if true {
		encoding.Encode(sliceA, sliceC) //@ diag(`overlapping dst and src`)
	}
	sliceD := sliceA[1:]
	sliceE := sliceA[1:]
	if true {
		encoding.Encode(sliceD, sliceE) //@ diag(`overlapping dst and src`)
	}
}