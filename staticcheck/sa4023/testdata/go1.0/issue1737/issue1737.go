// This tests that MaybeNilGlobal gets propagated correctly for the interface
// value's inner nilness.

package main

import (
	"errors"
	"log"
)

var errSentinal = errors.New("error")

func f1() error {
	return errSentinal
}

func f2() error {
	for {
		if err := f1(); err != nil {
			return err
		}
	}
}

func main() {
	if err := f2(); err != nil {
		log.Fatal(err)
	}
}
