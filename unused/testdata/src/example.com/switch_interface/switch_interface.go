package pkg

type t struct{} //@ used("t", true)

func (t) fragment() {} //@ used("fragment", true)

func fn() bool { //@ used("fn", true)
	var v interface{} = t{}  //@ used("v", true)
	switch obj := v.(type) { //@ used("obj", true)
	case interface {
		fragment() //@ used("fragment", true)
	}:
		obj.fragment()
	}
	return false
}

var x = fn() //@ used("x", true)
var _ = x    //@ used("_", true)
