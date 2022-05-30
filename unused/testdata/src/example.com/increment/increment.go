package pkg

type T struct { //@ used("T", true), used_test("T", true)
	// Writing to fields uses them
	f int //@ used("f", true), used_test("f", true)
}

// Not used, v is only written to
var v int //@ used("v", false), used_test("v", false)

func Foo() { //@ used("Foo", true), used_test("Foo", true)
	var x T //@ used("x", true), used_test("x", true)
	x.f++
	v++
}
