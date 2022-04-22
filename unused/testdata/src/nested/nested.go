package pkg

type t1 struct{} //@ used(false)

func (t1) fragment() {} //@ used(false)

func fn1() bool { //@ used(false)
	var v interface{} = t1{}
	switch obj := v.(type) {
	case interface {
		fragment()
	}:
		obj.fragment()
	}
	return false
}

type t2 struct{} //@ used(true)

func (t2) fragment() {} //@ used(true)

func Fn() bool { //@ used(true)
	var v interface{} = t2{}
	switch obj := v.(type) {
	case interface {
		fragment() //@ used(true)
	}:
		obj.fragment()
	}
	return false
}
