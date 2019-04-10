package pkg

import _ "unsafe"

//other:directive
//go:linkname ol other4

//go:linkname foo other1
func foo() {}

//go:linkname bar other2
var bar int

var (
	baz int // MATCH "baz is unused"
	//go:linkname qux other3
	qux int
)

//go:linkname fisk other3
var (
	fisk int
)

var ol int

//go:linkname doesnotexist other5
