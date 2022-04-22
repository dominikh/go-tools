package pkg

import "math"

func fn1() {
	var (
		u8  uint8
		u16 uint16
		u   uint

		i8  int8
		i16 int16
		i   int
	)

	_ = u8 > math.MaxUint8  //@ diag(`no value of type uint8 is greater than math.MaxUint8`)
	_ = u8 >= math.MaxUint8 //@ diag(`no value of type uint8 is greater than math.MaxUint8`)
	_ = u8 >= 0             //@ diag(`every value of type uint8 is >= 0`)
	_ = u8 <= math.MaxUint8 //@ diag(`every value of type uint8 is <= math.MaxUint8`)
	_ = u8 > 0
	_ = u8 >= 1
	_ = u8 < math.MaxUint8
	_ = u8 < 0 //@ diag(`no value of type uint8 is less than 0`)
	_ = u8 <= 0
	_ = 0 > u8 //@ diag(`no value of type uint8 is less than 0`)
	_ = 0 >= u8

	_ = u16 > math.MaxUint8
	_ = u16 > math.MaxUint16 //@ diag(`no value of type uint16 is greater than math.MaxUint16`)
	_ = u16 <= math.MaxUint8
	_ = u16 <= math.MaxUint16 //@ diag(`every value of type uint16 is <= math.MaxUint16`)

	_ = u > math.MaxUint32

	_ = i8 > math.MaxInt8 //@ diag(`no value of type int8 is greater than math.MaxInt8`)
	_ = i16 > math.MaxInt8
	_ = i16 > math.MaxInt16 //@ diag(`no value of type int16 is greater than math.MaxInt16`)
	_ = i > math.MaxInt32
	_ = i8 < 0
	_ = i8 <= math.MinInt8 //@ diag(`no value of type int8 is less than math.MinInt8`)
	_ = i8 < math.MinInt8  //@ diag(`no value of type int8 is less than math.MinInt8`)
	_ = i8 >= math.MinInt8 //@ diag(`every value of type int8 is >= math.MinInt8`)
}
