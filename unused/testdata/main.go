package main

func Fn1() {}
func Fn2() {} // MATCH Fn2

const X = 1 // MATCH X

var Y = 2 // MATCH Y

type Z struct{} // MATCH Z

func main() {
	Fn1()
}
