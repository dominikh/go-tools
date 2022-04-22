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
	var _ int = gen1()           //@ diag(`could omit type int`)
	var a int = Y                //@ diag(`could omit type int`)
	var b int = 1                //@ diag(`could omit type int`)
	var c int = 1.0              // different default type
	var d MyInt = 1              // different default type
	var e io.ReadCloser = gen2() //@ diag(`could omit type io.ReadCloser`)
	var f io.Reader = gen2()     // different interface type
	var g float64 = math.Pi      //@ diag(`could omit type float64`)
	var h bool = true            //@ diag(`could omit type bool`)
	var i string = ""            //@ diag(`could omit type string`)
	var j MyInt = gen3()         //@ diag(`could omit type MyInt`)
	var k uint8 = Y              // different default type on constant
	var l uint8 = (Y + Y) / 2    // different default type on rhs
	var m int = (Y + Y) / 2      //@ diag(`could omit type int`)

	_, _, _, _, _, _, _, _, _, _, _, _, _ = a, b, c, d, e, f, g, h, i, j, k, l, m
}
