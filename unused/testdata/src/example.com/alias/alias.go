package main

import "net/http"

type t1 struct{} //@ used("t1", true)
type t2 struct{} //@ used("t2", false)
type t3 struct{} //@ used("t3", true)

type alias1 = t1  //@ used("alias1", true)
type alias2 = t2  //@ used("alias2", false)
type alias3 = t3  //@ used("alias3", false)
type alias4 = int //@ used("alias4", false)

func main() { //@ used("main", true)
	var _ alias1 //@ used("_", true)
	var _ t3     //@ used("_", true)
}

type t4 struct { //@ used("t4", true)
	x int //@ used("x", true)
}

func (t4) foo() {} //@ used("foo", true)

//lint:ignore U1000 alias5 is ignored, which also ignores t4
type alias5 = t4 //@ used("alias5", true)

//lint:ignore U1000 alias6 is ignored, and we don't incorrectly try to include http.Server's fields and methods in the graph
type alias6 = http.Server //@ used("alias6", true)

//lint:ignore U1000 aliases don't have to be to named types
type alias7 = struct { //@ used("alias7", true)
	x int //@ used("x", true)
}
