// Package pkg ...
package pkg

import (
	"errors"
	"fmt"
)

var (
	foo                   = errors.New("") // MATCH "error var foo should have name of the form errFoo"
	errBar                = errors.New("")
	qux, fisk, errAnother = errors.New(""), errors.New(""), errors.New("")
	abc                   = fmt.Errorf("") // MATCH "error var abc should have name of the form errFoo"

	errAbc = fmt.Errorf("")
)

var wrong = errors.New("") // MATCH "error var wrong should have name of the form errFoo"

var result = fn()

func fn() error { return nil }

// MATCH:12 "error var qux should have name of the form errFoo"
// MATCH:12 "error var fisk should have name of the form errFoo"
