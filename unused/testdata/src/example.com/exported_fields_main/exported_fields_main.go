package main

type t1 struct { //@ used("t1", true)
	F1 int //@ used("F1", true)
}

type T2 struct { //@ used("T2", true)
	F2 int //@ used("F2", true)
}

func init() { //@ used("init", true)
	_ = t1{}
	_ = T2{}
}
