package pkg

import _ "unsafe"

//go:linkname foo bar
func foo() {}
