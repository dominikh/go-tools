package pkg

type I interface { //@ used(true)
	foo() //@ used(true)
}

type T struct{} //@ used(true)

func (T) foo() {} //@ used(true)
func (T) bar() {} //@ used(false)

var _ struct {
	T //@ used(true)
}
