package pkg

// Used because v2 is a sink
var v2 int //@ used_test("v2", true)

func Bar() { //@ used_test("Bar", true)
	v2++
}
