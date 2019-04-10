package pkg

type t struct{} // MATCH /t is unused/

func (t) fragment() {}

func fn() bool { // MATCH /fn is unused/
	var v interface{} = t{}
	switch obj := v.(type) {
	case interface {
		fragment()
	}:
		obj.fragment()
	}
	return false
}
