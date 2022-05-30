package main

type t1 struct{} //@ used("t1", true)
type t2 struct{} //@ used("t2", true)

func (t1) foo(arg *t2) {} //@ used("foo", true), used("arg", true)

func init() { //@ used("init", true)
	t1{}.foo(nil)
}
