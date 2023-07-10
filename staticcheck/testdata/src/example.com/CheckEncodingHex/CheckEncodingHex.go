package pkg

import (
	"encoding/hex"
	"log"
)

func fn() {
	log.Println(hex.Encode(nil, nil))
	log.Println(hex.Encode(make([]byte, 0), nil))
	sliceA := make([]byte, 8)
	sliceB := make([]byte, 8)
	log.Println(hex.Encode(sliceA, sliceB))
	log.Println(hex.Encode(sliceA, sliceA)) //@ diag(`overlapping dst and src`)
	log.Println(hex.Encode(sliceA[1:], sliceA[2:]))
	log.Println(hex.Encode(sliceA[1:], sliceA[1:])) //@ diag(`overlapping dst and src`)
	sliceC := sliceA
	log.Println(hex.Encode(sliceA, sliceC)) //@ diag(`overlapping dst and src`)
	if true {
		log.Println(hex.Encode(sliceA, sliceC)) //@ diag(`overlapping dst and src`)
	}
	sliceD := sliceA[1:]
	sliceE := sliceA[1:]
	if true {
		log.Println(hex.Encode(sliceD, sliceE)) //@ diag(`overlapping dst and src`)
	}
}
