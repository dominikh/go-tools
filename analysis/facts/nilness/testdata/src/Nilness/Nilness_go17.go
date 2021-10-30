//go:build go1.17
// +build go1.17

package pkg

func fn21() *[5]int { // want fn21:`never returns nil: \[never\]`
	var x []int
	return (*[5]int)(x)
}

func fn22() *[0]int {
	var x []int
	return (*[0]int)(x)
}

func fn23() *[5]int { // want fn23:`never returns nil: \[never\]`
	var x []int
	type T [5]int
	ret := (*T)(x)
	return (*[5]int)(ret)
}

func fn24() *[0]int {
	var x []int
	type T [0]int
	ret := (*T)(x)
	return (*[0]int)(ret)
}

func fn25() *[5]int { // want fn25:`never returns nil: \[never\]`
	var x []int
	type T *[5]int
	return (T)(x)
}

func fn26() *[0]int {
	var x []int
	type T *[0]int
	return (T)(x)
}
