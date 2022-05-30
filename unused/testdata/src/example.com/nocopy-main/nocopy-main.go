package main

type myNoCopy1 struct{}   //@ used("myNoCopy1", true)
type myNoCopy2 struct{}   //@ used("myNoCopy2", true)
type wrongLocker struct{} //@ used("wrongLocker", false)
type someStruct struct {  //@ used("someStruct", false)
	x int //@ quiet("x")
}

func (myNoCopy1) Lock()      {} //@ used("Lock", true)
func (recv myNoCopy2) Lock() {} //@ used("Lock", true), used("recv", true)
func (wrongLocker) lock()    {} //@ used("lock", false)
func (wrongLocker) unlock()  {} //@ used("unlock", false)
func (someStruct) Lock()     {} //@ used("Lock", false)

type T struct { //@ used("T", true)
	noCopy1 myNoCopy1   //@ used("noCopy1", true)
	noCopy2 myNoCopy2   //@ used("noCopy2", true)
	field1  someStruct  //@ used("field1", false)
	field2  wrongLocker //@ used("field2", false)
	field3  int         //@ used("field3", false)
}

func main() { //@ used("main", true)
	_ = T{}
}
