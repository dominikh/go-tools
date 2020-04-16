package main

type t1 struct{} // used
type t2 struct{} // unused
type t3 struct{} // used

type alias1 = t1  // used
type alias2 = t2  // unused
type alias3 = t3  // used
type alias4 = int // used

func main() { // used
	var _ alias1
	var _ t3
}
