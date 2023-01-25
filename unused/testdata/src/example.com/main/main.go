package main

func Fn1() {} //@ used(true)
func Fn2() {} //@ used(true)
func fn3() {} //@ used(false)

const X = 1 //@ used(true)

var Y = 2 //@ used(true)

type Z struct{} //@ used(true)

func main() { //@ used(true)
	Fn1()
}
