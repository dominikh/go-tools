package main

func Fn1() {} //@ used("Fn1", true)
func Fn2() {} //@ used("Fn2", true)
func fn3() {} //@ used("fn3", false)

const X = 1 //@ used("X", true)

var Y = 2 //@ used("Y", true)

type Z struct{} //@ used("Z", true)

func main() { //@ used("main", true)
	Fn1()
}
