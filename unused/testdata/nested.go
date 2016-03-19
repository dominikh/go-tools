package pkg

type t struct{} // MATCH t

func (t) fragment() {}

func fn() bool { // MATCH fn
	var v interface{} = t{}
	switch obj := v.(type) {
	// XXX it shouldn't report fragment(), because fn is unused
	case interface {
		fragment() // MATCH fragment
	}:
		obj.fragment()
	}
	return false
}
