package pkg

type t struct{} //@ used(true)

func (t) fn1() {} //@ used(true)
func (t) fn2() {} //@ used(true)
func fn1()     {} //@ used(true)
func fn2()     {} //@ used(true)

func Fn() { //@ used(true)
	var v t
	defer fn1()
	defer v.fn1()
	go fn2()
	go v.fn2()
}
