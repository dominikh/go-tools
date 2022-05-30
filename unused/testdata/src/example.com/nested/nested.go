package pkg

type t1 struct{} //@ used("t1", false)

func (t1) fragment() {} //@ used("fragment", false)

func fn1() bool { //@ used("fn1", false)
	var v interface{} = t1{} //@ quiet("v")
	switch obj := v.(type) { //@ quiet("obj")
	case interface {
		fragment() //@ quiet("fragment")
	}:
		obj.fragment()
	}
	return false
}

type t2 struct{} //@ used("t2", true)

func (t2) fragment() {} //@ used("fragment", true)

func Fn() bool { //@ used("Fn", true)
	var v interface{} = t2{} //@ used("v", true)
	switch obj := v.(type) { //@ used("obj", true)
	case interface {
		fragment() //@ used("fragment", true)
	}:
		obj.fragment()
	}
	return false
}

func Fn2() bool { //@ used("Fn2", true)
	var v interface{} = t2{} //@ used("v", true)
	switch obj := v.(type) { //@ used("obj", true)
	case interface {
		fragment() //@ used("fragment", true)
	}:
		_ = obj
	}
	return false
}
