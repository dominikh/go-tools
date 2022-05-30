package pkg

type t struct{} //@ used("t", true)

func (t) fn1() {} //@ used("fn1", true)
func (t) fn2() {} //@ used("fn2", true)
func fn1()     {} //@ used("fn1", true)
func fn2()     {} //@ used("fn2", true)

func Fn() { //@ used("Fn", true)
	var v t //@ used("v", true)
	defer fn1()
	defer v.fn1()
	go fn2()
	go v.fn2()
}
