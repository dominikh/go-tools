// Package pkg ...
package pkg

import (
	fmt "fmt" //@ diag(`package "fmt" is imported with a redundant alias`)
	"math"

	adiffname "example.com/diffname"
	samename "example.com/samename" //@ diag(`package "example.com/samename" is imported with a redundant alias`)
)

var (
	_ = fmt.Println
	_ = math.E
	_ = adiffname.I
	_ = samename.I
)
