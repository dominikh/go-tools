package main

import (
	"testing"
)

type t1 struct{} //@ used_test(true)

func TestFoo(t *testing.T) { //@ used_test(true)
	_ = t1{}
}
