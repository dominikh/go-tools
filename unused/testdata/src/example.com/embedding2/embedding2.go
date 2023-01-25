package main

type AA interface { //@ used(true)
	A() //@ used(true)
}

type BB interface { //@ used(true)
	AA
}

type CC interface { //@ used(true)
	BB
	C() //@ used(true)
}

func c(cc CC) { //@ used(true)
	cc.A()
}

type z struct{} //@ used(true)

func (z) A() {} //@ used(true)
func (z) B() {} //@ used(true)
func (z) C() {} //@ used(true)

func main() { //@ used(true)
	c(z{})
}
