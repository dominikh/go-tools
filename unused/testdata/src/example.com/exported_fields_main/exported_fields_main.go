package main

type t1 struct { //@ used(true)
	F1 int //@ used(true)
}

type T2 struct { //@ used(true)
	F2 int //@ used(true)
}

func init() { //@ used(true)
	_ = t1{}
	_ = T2{}
}
