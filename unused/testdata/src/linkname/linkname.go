package pkg

import _ "unsafe"

//other:directive
//go:linkname ol other4

//go:linkname foo other1
func foo() {} //@ used(true)

//go:linkname bar other2
var bar int //@ used(true)

var (
	baz int //@ used(false)
	//go:linkname qux other3
	qux int //@ used(true)
)

//go:linkname fisk other3
var (
	fisk int //@ used(true)
)

var ol int //@ used(true)

//go:linkname doesnotexist other5
