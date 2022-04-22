// Package pkg ...
package pkg

import (
	"errors"
	"fmt"
)

var (
	foo                   = errors.New("") //@ diag(`error var foo should have name of the form errFoo`)
	errBar                = errors.New("")
	qux, fisk, errAnother = errors.New(""), errors.New(""), errors.New("") //@ diag(`error var qux should have name of the form errFoo`), diag(`error var fisk should have name of the form errFoo`)
	abc                   = fmt.Errorf("")                                 //@ diag(`error var abc should have name of the form errFoo`)

	errAbc = fmt.Errorf("")
)

var wrong = errors.New("") //@ diag(`error var wrong should have name of the form errFoo`)

var result = fn()

func fn() error { return nil }
