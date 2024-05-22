package pkg

import pkg "example.com/CheckDeprecated.assist"

func tpFn() {
	var x pkg.S[int]
	x.Foo()
	x.Bar() //@ diag(`deprecated`)
	x.Baz() //@ diag(`deprecated`)
	x.Qux()
	_ = x.Field1
	_ = x.Field2 // This should be flagged, but see issue 1215
}
