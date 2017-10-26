package pkg

func fn1() {} // MATCH "test problem"

//lint:ignore TEST1000 This should be ignored, because ...
func fn2() {}

//lint:ignore TEST1000
func fn3() {} // MATCH "test problem"

// MATCH:8 "malformed linter directive"
