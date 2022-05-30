package pkg

type I interface { //@ used("I", true)
	foo() //@ used("foo", true)
}

type T struct{} //@ used("T", true)

func (T) foo() {} //@ used("foo", true)
func (T) bar() {} //@ used("bar", false)

var _ struct { //@ used("_", true)
	T //@ used("T", true)
}
