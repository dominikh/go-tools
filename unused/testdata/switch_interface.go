package pkg

type t struct{}

func (t) fragment() {}

func fn() bool {
	var v interface{} = t{}
	switch v.(type) {
	case interface {
		fragment()
	}:
	}
	return false
}

var x = fn()
var _ = x
