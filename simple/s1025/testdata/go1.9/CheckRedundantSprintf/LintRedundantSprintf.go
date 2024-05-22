package pkg

import "fmt"

type T6 string
type T9 string

type MyByte byte

type Alias = byte
type T13 = []byte
type T14 = []Alias

type T15 = string
type T16 = T9
type T17 = T6

func (T6) String() string { return "" }

func (T9) Format(f fmt.State, c rune) {}

func fn() {
	var t12 []Alias
	var t13 T13
	var t14 T14
	var t15 T15
	var t16 T16
	var t17 T17

	_ = fmt.Sprintf("%s", t12) //@ diag(`underlying type is a slice of bytes`)
	_ = fmt.Sprintf("%s", t13) //@ diag(`underlying type is a slice of bytes`)
	_ = fmt.Sprintf("%s", t14) //@ diag(`underlying type is a slice of bytes`)
	_ = fmt.Sprintf("%s", t15) //@ diag(`is already a string`)
	_ = fmt.Sprintf("%s", t17) //@ diag(`should use String() instead of fmt.Sprintf`)

	// don't simplify types that implement fmt.Formatter
	_ = fmt.Sprintf("%s", t16)
}
