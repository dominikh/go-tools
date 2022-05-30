package main

import (
	"testing"
)

type t1 struct{} //@ used_test("t1", true)

func TestFoo(t *testing.T) { //@ used_test("TestFoo", true), used_test("t", true)
	_ = t1{}
}
