package pkg

import (
	"io"
	"math"
)

type MyInt int

const X int = 1
const Y = 1

func gen1() int           { return 0 }
func gen2() io.ReadCloser { return nil }
func gen3() MyInt         { return 0 }

// don't flag global variables
var a int = gen1()

func fn() {
	var _ int = gen1()           // don't flag blank identifier
	var a int = Y                // don't flag named untyped constants
	var b int = 1                //@ diag(`should omit type int`)
	var c int = 1.0              // different default type
	var d MyInt = 1              // different default type
	var e io.ReadCloser = gen2() //@ diag(`should omit type io.ReadCloser`)
	var f io.Reader = gen2()     // different interface type
	var g float64 = math.Pi      // don't flag named untyped constants
	var h bool = true            //@ diag(`should omit type bool`)
	var i string = ""            //@ diag(`should omit type string`)
	var j MyInt = gen3()         //@ diag(`should omit type MyInt`)
	var k uint8 = Y              // different default type on constant
	var l uint8 = (Y + Y) / 2    // different default type on rhs
	var m int = (Y + Y) / 2      // complex expression

	_, _, _, _, _, _, _, _, _, _, _, _, _ = a, b, c, d, e, f, g, h, i, j, k, l, m
}
