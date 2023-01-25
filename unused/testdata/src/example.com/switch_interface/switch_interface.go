package pkg

type t struct{} //@ used(true)

func (t) fragment() {} //@ used(true)

func fn() bool { //@ used(true)
	var v interface{} = t{}
	switch obj := v.(type) {
	case interface {
		fragment() //@ used(true)
	}:
		obj.fragment()
	}
	return false
}

var x = fn() //@ used(true)
var _ = x
