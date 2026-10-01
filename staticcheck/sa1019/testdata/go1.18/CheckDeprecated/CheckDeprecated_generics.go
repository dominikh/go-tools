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

func tpCompositeLit() {
	// Keys in composite literals of instantiated generic types used to crash
	// SA1019 with "unsupported selector".
	_ = pkg.S2[int]{
		Field1: 1,
		Field2: "x", //@ diag(`(example.com/CheckDeprecated.assist.S2[int]).Field2 is deprecated: don't use me`)
	}
	_ = pkg.S3[int, bool]{
		Field1: 1,
		Field3: "x", //@ diag(`(example.com/CheckDeprecated.assist.S3[int, bool]).Field3 is deprecated: don't use me`)
	}
}
