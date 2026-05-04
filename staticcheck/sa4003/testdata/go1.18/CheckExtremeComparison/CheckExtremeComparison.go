package pkg

import "math"

func fn1[T uint8 | uint16](x T) {
	_ = x >= 0 //@ diag(`every value of type T is >= 0`)
	_ = x < 0  //@ diag(`no value of type T is less than 0`)

	// No diagnostic, T has multiple sizes
	_ = x > math.MaxUint8
}

func fn2[T uint8 | int8](x T) {
	// No diagnostics, no common signedness or size
	_ = x >= 0
	_ = x < 0
}

func fn3[T uint8](x T) {
	_ = x >= 0            //@ diag(`every value of type T is >= 0`)
	_ = x < 0             //@ diag(`no value of type T is less than 0`)
	_ = x > math.MaxUint8 //@ diag(`no value of type T is greater than math.MaxUint8`)
}

func fn4[T ~uint8](x T) {
	_ = x >= 0            //@ diag(`every value of type T is >= 0`)
	_ = x < 0             //@ diag(`no value of type T is less than 0`)
	_ = x > math.MaxUint8 //@ diag(`no value of type T is greater than math.MaxUint8`)
}

type (
	U1 uint8
	U2 uint8
)

func fn5[T U1 | U2](x T) {
	_ = x >= 0            //@ diag(`every value of type T is >= 0`)
	_ = x < 0             //@ diag(`no value of type T is less than 0`)
	_ = x > math.MaxUint8 //@ diag(`no value of type T is greater than math.MaxUint8`)
}

type S[T uint8 | uint16] struct {
	f T
}

func (s S[T]) fn6() {
	_ = s.f >= 0 //@ diag(`every value of type T is >= 0`)
	_ = s.f < 0  //@ diag(`no value of type T is less than 0`)

	// No diagnostic, T has multiple sizes
	_ = s.f > math.MaxUint8
}

func (s S[T]) fn7(x T) {
	_ = x >= 0 //@ diag(`every value of type T is >= 0`)
	_ = x < 0  //@ diag(`no value of type T is less than 0`)

	// No diagnostic, T has multiple sizes
	_ = x > math.MaxUint8
}
