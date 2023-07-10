package pkg

import (
	"encoding/ascii85"
	"log"
)

func fn() {
	log.Println(ascii85.Encode(nil, nil))
	log.Println(ascii85.Encode(make([]byte, 0), nil))
	sliceA := make([]byte, 8)
	sliceB := make([]byte, 8)
	log.Println(ascii85.Encode(sliceA, sliceB))
	log.Println(ascii85.Encode(sliceA, sliceA)) //@ diag(`overlapping dst and src`)
	log.Println(ascii85.Encode(sliceA[1:], sliceA[2:]))
	log.Println(ascii85.Encode(sliceA[1:], sliceA[1:])) //@ diag(`overlapping dst and src`)
	sliceC := sliceA
	log.Println(ascii85.Encode(sliceA, sliceC)) //@ diag(`overlapping dst and src`)
	if true {
		log.Println(ascii85.Encode(sliceA, sliceC)) //@ diag(`overlapping dst and src`)
	}
	sliceD := sliceA[1:]
	sliceE := sliceA[1:]
	if true {
		log.Println(ascii85.Encode(sliceD, sliceE)) //@ diag(`overlapping dst and src`)
	}
}
