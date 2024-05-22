package pkg

import "fmt"

func fn() {
	type ByteSlice []byte
	type Alias1 = []byte
	type Alias2 = ByteSlice
	type Alias3 = byte
	type Alias4 = []Alias3

	fmt.Print(Alias1{}) //@ diag(`could convert argument to string`)
	fmt.Print(Alias2{}) //@ diag(`could convert argument to string`)
	fmt.Print(Alias4{}) //@ diag(`could convert argument to string`)
	fmt.Print()
	fmt.Fprint(nil)
}
