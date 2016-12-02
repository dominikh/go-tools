package pkg

import "strings"

func fn() {
	strings.Replace("", "", "", 1) // MATCH /is a pure function but its return value is ignored/
	foo(1, 2)                      // MATCH /is a pure function but its return value is ignored/
	bar(1, 2)
}

func foo(a, b int) int { return a + b }
func bar(a, b int) int {
	println(a + b)
	return a + b
}
