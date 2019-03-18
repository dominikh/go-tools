package pkg

// the line directive should not affect the line ignores

//line random-file:1
func fn1() {} // MATCH "test problem"

//lint:ignore TEST1000 This should be ignored, because ...
//lint:ignore XXX1000 Testing that multiple linter directives work correctly
func fn2() {}

//lint:ignore TEST1000
func fn3() {} // MATCH "test problem"

//lint:ignore TEST1000 ignore
func fn4() {
	//lint:ignore TEST1000 ignore
	var _ int
}

// MATCH:12 "malformed linter directive"
// MATCH:17 "this linter directive didn't match anything"
