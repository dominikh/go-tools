package main

type state func() state //@ used(true)

func a() state { //@ used(true)
	return a
}

func main() { //@ used(true)
	st := a
	_ = st()
}

type t1 struct{} //@ used(false)
type t2 struct{} //@ used(true)
type t3 struct{} //@ used(true)

func fn1() t1     { return t1{} } //@ used(false)
func fn2() (x t2) { return }      //@ used(true)
func fn3() *t3    { return nil }  //@ used(true)

func fn4() { //@ used(true)
	const x = 1  //@ used(true)
	const y = 2  //@ used(false)
	type foo int //@ used(false)
	type bar int //@ used(true)

	_ = x
	_ = bar(0)
}

func init() { //@ used(true)
	fn2()
	fn3()
	fn4()
}
