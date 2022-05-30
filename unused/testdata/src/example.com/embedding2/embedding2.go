package main

type AA interface { //@ used("AA", true)
	A() //@ used("A", true)
}

type BB interface { //@ used("BB", true)
	AA
}

type CC interface { //@ used("CC", true)
	BB
	C() //@ used("C", true)
}

func c(cc CC) { //@ used("c", true), used("cc", true)
	cc.A()
}

type z struct{} //@ used("z", true)

func (z) A() {} //@ used("A", true)
func (z) B() {} //@ used("B", true)
func (z) C() {} //@ used("C", true)

func main() { //@ used("main", true)
	c(z{})
}
