package pkg

import "testing"

func TestFn(t *testing.T) { //@ used_test("TestFn", true), used_test("t", true)
	fn()
}
