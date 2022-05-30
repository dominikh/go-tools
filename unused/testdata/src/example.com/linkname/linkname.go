package pkg

import _ "unsafe"

//other:directive
//go:linkname ol other4

//go:linkname foo other1
func foo() {} //@ used("foo", true)

//go:linkname bar other2
var bar int //@ used("bar", true)

var (
	baz int //@ used("baz", false)
	//go:linkname qux other3
	qux int //@ used("qux", true)
)

//go:linkname fisk other3
var (
	fisk int //@ used("fisk", true)
)

var ol int //@ used("ol", true)

//go:linkname doesnotexist other5
