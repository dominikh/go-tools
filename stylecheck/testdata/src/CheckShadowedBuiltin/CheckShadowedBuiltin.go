// Package pkg ...
package pkg

import "fmt"

type X struct {
	len int
}

func cap() {} // MATCH "shadows built-in"

type Y struct{}

func (Y) len() {}

func fn(len int) { // MATCH "shadows built-in"
	fmt.Println(len)
}

func fn2() {
	var len int // MATCH "shadows built-in"
	_ = len
}

func fn3(x []string) {
	_ = len(x)
}
