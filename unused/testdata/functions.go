package main

type state func() state

func a() state {
	return a
}

func main() {
	st := a
	_ = st()
}

type t1 struct{} // MATCH t1
type t2 struct{}
type t3 struct{}

func fn1() t1     { return t1{} } // MATCH fn1
func fn2() (x t2) { return }
func fn3() *t3    { return nil }

func init() {
	fn2()
	fn3()
}
