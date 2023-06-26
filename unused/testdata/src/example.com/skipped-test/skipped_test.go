package pkg

// https://staticcheck.dev/issues/633

import "testing"

func test() { //@ used_test("test", true)
}

func TestSum(t *testing.T) { //@ used_test("TestSum", true), used_test("t", true)
	t.Skip("skipping for test")
	test()
}
