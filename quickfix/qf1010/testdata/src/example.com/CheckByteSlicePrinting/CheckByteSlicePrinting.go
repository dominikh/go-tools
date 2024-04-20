package pkg

import "fmt"

type Stringable []byte

func (*Stringable) String() string { return "" }

func fn() {
	type ByteSlice []byte
	type Alias1 = []byte
	type Alias2 = ByteSlice
	type Alias3 = byte
	type Alias4 = []Alias3
	var b1 []byte
	var b2 ByteSlice
	var b3 *Stringable
	var b4 Stringable

	var s string
	fmt.Print(1, b1, 2, []byte(""), b2, s)       //@ diag(`could convert argument to string`), diag(`could convert argument to string`), diag(`could convert argument to string`)
	fmt.Print(Alias1{})                          //@ diag(`could convert argument to string`)
	fmt.Print(Alias2{})                          //@ diag(`could convert argument to string`)
	fmt.Print(Alias4{})                          //@ diag(`could convert argument to string`)
	fmt.Fprint(nil, 1, b1, 2, []byte(""), b2, s) //@ diag(`could convert argument to string`), diag(`could convert argument to string`), diag(`could convert argument to string`)
	fmt.Print()
	fmt.Fprint(nil)

	fmt.Println(b3)
	fmt.Println(b4) //@ diag(`could convert argument to string`)
}
