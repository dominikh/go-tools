// +build go1.15

package pkg

import "strconv"

func fn() {
	strconv.ParseComplex("", 32) //@ diag(`'bitSize' argument is invalid, must be either 64 or 128`)
	strconv.ParseComplex("", 64)
	strconv.ParseComplex("", 128)
	strconv.ParseComplex("", 256) //@ diag(`'bitSize' argument is invalid, must be either 64 or 128`)

	strconv.FormatComplex(0, 'e', 0, 32) //@ diag(`'bitSize' argument is invalid, must be either 64 or 128`)
	strconv.FormatComplex(0, 'e', 0, 64)
	strconv.FormatComplex(0, 'e', 0, 128)
	strconv.FormatComplex(0, 'e', 0, 256) //@ diag(`'bitSize' argument is invalid, must be either 64 or 128`)
	strconv.FormatComplex(0, 'j', 0, 64)  //@ diag(`'fmt' argument is invalid: unknown format 'j'`)
}
