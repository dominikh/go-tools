package main

type t1 struct{} //@ used(true)
type t2 struct{} //@ used(true)

func (t1) foo(arg *t2) {} //@ used(true)

func init() { //@ used(true)
	t1{}.foo(nil)
}
