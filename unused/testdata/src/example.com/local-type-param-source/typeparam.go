package tparamsource

// https://staticcheck.dev/issues/1282

import (
	"testing"

	tparamsink "example.com/local-type-param-sink"
)

func TestFoo(t *testing.T) { //@ used("TestFoo", true), used("t", true)
	type EmptyStruct struct{} //@ used("EmptyStruct", true)
	_ = tparamsink.TypeOfType[EmptyStruct]()
}
