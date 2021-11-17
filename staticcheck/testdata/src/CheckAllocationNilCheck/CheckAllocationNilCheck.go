package pkg

type T struct {
	Field func()
}

func (T) Method() {}

func gen() *int { return nil }

func fn1() {
	var a *int
	b := new(int)
	c := make([]byte, 0)
	var t T
	var pt *T
	d := t.Field
	e := t.Method
	f := &t.Field
	g := fn1
	h := &T{}
	i := gen()
	j := func() {}
	k := make(map[string]int)
	var slice []byte
	l := slice[:0]
	var m []byte
	if true {
		m = []byte{}
	} else {
		m = []byte{}
	}
	n := m[:0]
	o := &pt.Field

	if a != nil {
	}
	if b != nil { // want `always true`
	}
	if b == nil { // want `never true`
	}
	if c != nil { // want `always true`
	}
	if d != nil { // field value could be anything
	}
	if e != nil { // want `contains a function`
	}
	if f != nil { // want `always true`
	}
	if g != nil { // want `contains a function`
	}
	if h != nil { // want `always true`
	}
	if &a != nil { // already flagged by SA4022
	}
	if (&T{}).Method != nil { // want `always true`
	}
	if (&T{}) != nil { // already flagged by SA4022
	}
	if i != nil { // just a function return value
	}
	if fn1 != nil { // want `functions are never nil`
	}
	if j != nil { // want `contains a function`
	}
	if k != nil { // want `always true`
	}
	if l != nil { // slicing a nil slice yields nil
	}
	if m != nil { // want `always true`
	}
	if n != nil { // want `always true`
	}
	if o != nil { // in &pt.Field, pt might be nil
	}
}

func fn2() {
	x := new(int)
	if true {
		x = nil
	}
	if x != nil {
	}
}
