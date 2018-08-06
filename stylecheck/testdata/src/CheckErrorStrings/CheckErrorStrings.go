// Package pkg ...
package pkg

import "errors"

func fn() {
	errors.New("a perfectly fine error")
	errors.New("Not a great error")       // MATCH "error strings should not be capitalized"
	errors.New("also not a great error.") // MATCH "error strings should not end with punctuation or a newline"
	errors.New("URL is okay")
	errors.New("SomeFunc is okay")
	errors.New("URL is okay, but the period is not.") // MATCH "error strings should not end with punctuation or a newline"
}
