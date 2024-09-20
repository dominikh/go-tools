//go:build go1.23

package pkg

import "structs"

// hostLayout isn't used because the fields of hlt5 aren't marked used because
// the hostLayout named type doesn't trigger the structs.HostLayout logic.
type hostLayout structs.HostLayout        //@ used("hostLayout", false)
type hostLayoutAlias = structs.HostLayout //@ used("hostLayoutAlias", true)

type hlt1 struct { //@ used("hlt1", true)
	_     structs.HostLayout //@ used("_", true)
	hlf11 int                //@ used("hlf11", true)
}

type hlt2 struct { //@ used("hlt2", true)
	structs.HostLayout     //@ used("HostLayout", true)
	hlf21              int //@ used("hlf21", true)
}

type hlt3 struct { //@ used("hlt3", true)
	// Aliases of structs.HostLayout do mark fields used.
	hostLayoutAlias     //@ used("hostLayoutAlias", true)
	hlf31           int //@ used("hlf31", true)
}

type hlt4 struct { //@ used("hlt4", true)
	// Embedding a struct that itself has a structs.HostLayout field doesn't
	// mark this struct's fields used.
	hlt3      //@ used("hlt3", false)
	hlf41 int //@ used("hlf41", false)
}

type hlt5 struct { //@ used("hlt5", true)
	// Named types with underlying type structs.HostLayout don't mark fields
	// used.
	hostLayout     //@ used("hostLayout", false)
	hlf51      int //@ used("hlf51", false)
}

type hlt6 struct { //@ used("hlt6", false)
	// The fields may be used, but the overall struct isn't.
	_     structs.HostLayout //@ quiet("_")
	hlf61 int                //@ quiet("hlf61")
}

type hlt7 struct { //@ used("hlt7", true)
	// Fields are used recursively, as they affect the layout
	_     structs.HostLayout //@ used("_", true)
	hlf71 [2]hlt7sub         //@ used("hlf71", true)
}
type hlt7sub struct { //@ used("hlt7sub", true)
	hlt7sub1 int //@ used("hlt7sub1", true)
}

var _ hlt1 //@ used("_", true)
var _ hlt2 //@ used("_", true)
var _ hlt3 //@ used("_", true)
var _ hlt4 //@ used("_", true)
var _ hlt5 //@ used("_", true)
var _ hlt7 //@ used("_", true)
