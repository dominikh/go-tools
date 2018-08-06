package pkg

// the line directive should not affect the line ignores

//line random-file:1
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

// MATCH:11 "malformed linter directive"
// MATCH:16 "this linter directive didn't match anything"
