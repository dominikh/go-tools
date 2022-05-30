package pkg

import (
	"io"
	"unsafe"
)

// https://staticcheck.io/issues/812

var (
	w  io.Writer          //@ used("w", true)
	sz = unsafe.Sizeof(w) //@ used("sz", true)
)

var _ = sz //@ used("_", true)

type t struct { //@ used("t", true)
	F int //@ used("F", true)
}

const S = unsafe.Sizeof(t{}) //@ used("S", true)
