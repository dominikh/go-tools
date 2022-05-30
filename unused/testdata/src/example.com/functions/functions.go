package main

type state func() state //@ used("state", true)

func a() state { //@ used("a", true)
	return a
}

func main() { //@ used("main", true)
	st := a //@ used("st", true)
	_ = st()
}

type t1 struct{} //@ used("t1", false)
type t2 struct{} //@ used("t2", true)
type t3 struct{} //@ used("t3", true)

func fn1() t1     { return t1{} } //@ used("fn1", false)
func fn2() (x t2) { return }      //@ used("fn2", true), used("x", true)
func fn3() *t3    { return nil }  //@ used("fn3", true)

func fn4() { //@ used("fn4", true)
	const x = 1  //@ used("x", true)
	const y = 2  //@ used("y", false)
	type foo int //@ used("foo", false)
	type bar int //@ used("bar", true)

	_ = x
	_ = bar(0)
}

func init() { //@ used("init", true)
	fn2()
	fn3()
	fn4()
}
