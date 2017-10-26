package pkg

func fn1() {} // MATCH "test problem"

//lint:ignore TEST1000 This should be ignored, because ...
func fn2() {}

//lint:ignore TEST1000
func fn3() {} // MATCH "test problem"

//lint:ignore TEST1000 ignore
func fn4() {
	//lint:ignore TEST1000 ignore
	var _ int
}

// MATCH:8 "malformed linter directive"
// MATCH:13 "this linter directive didn't match anything"
