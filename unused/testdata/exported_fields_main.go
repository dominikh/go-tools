package main

type t1 struct {
	F1 int // MATCH F1
}

type T2 struct {
	F2 int // MATCH F2
}

func init() {
	_ = t1{}
	_ = T2{}
}
