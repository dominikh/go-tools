package main

import "net/http"

type t1 struct{} //@ used(true)
type t2 struct{} //@ used(false)
type t3 struct{} //@ used(true)

type alias1 = t1  //@ used(true)
type alias2 = t2  //@ used(false)
type alias3 = t3  //@ used(true)
type alias4 = int //@ used(true)

func main() { //@ used(true)
	var _ alias1
	var _ t3
}

type t4 struct { //@ used(true)
	x int //@ used(true)
}

func (t4) foo() {} //@ used(true)

//lint:ignore U1000 alias5 is ignored, which also ignores t4
type alias5 = t4 //@ used(true)

//lint:ignore U1000 alias6 is ignored, and we don't incorrectly try to include http.Server's fields and methods in the graph
type alias6 = http.Server //@ used(true)

//lint:ignore U1000 aliases don't have to be to named types
type alias7 = struct { //@ used(true)
	x int //@ used(true)
}
