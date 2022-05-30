package bar

type myNoCopy1 struct{}    //@ used("myNoCopy1", true)
type myNoCopy2 struct{}    //@ used("myNoCopy2", true)
type stdlibNoCopy struct{} //@ used("stdlibNoCopy", true)
type locker struct{}       //@ used("locker", false)
type someStruct struct {   //@ used("someStruct", false)
	x int //@ quiet("x")
}

func (myNoCopy1) Lock()      {} //@ used("Lock", true)
func (recv myNoCopy2) Lock() {} //@ used("Lock", true), used("recv", true)
func (locker) Lock()         {} //@ used("Lock", false)
func (locker) Foobar()       {} //@ used("Foobar", false)
func (someStruct) Lock()     {} //@ used("Lock", false)

func (stdlibNoCopy) Lock()   {} //@ used("Lock", true)
func (stdlibNoCopy) Unlock() {} //@ used("Unlock", true)

type T struct { //@ used("T", true)
	noCopy1 myNoCopy1    //@ used("noCopy1", true)
	noCopy2 myNoCopy2    //@ used("noCopy2", true)
	noCopy3 stdlibNoCopy //@ used("noCopy3", true)
	field1  someStruct   //@ used("field1", false)
	field2  locker       //@ used("field2", false)
	field3  int          //@ used("field3", false)
}
