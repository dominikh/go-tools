package pkg

import (
	"errors"
	"os/exec"
)

type T struct{ x *int }

func notAStub() {}

func fn1() *int             { return nil }       // want fn1:`nilness: \[\{[^ ]+ AlwaysNil\}\]`
func fn2() (int, *int, int) { return 0, nil, 0 } // want fn2:`nilness: \[\{NeverNil NeverNil\} \{[^ ]+ AlwaysNil\} \{NeverNil NeverNil\}\]`

func fn3() (out1 int, out2 error) { notAStub(); return 0, nil } // want fn3:`nilness: \[\{[^}]+\} \{AlwaysNil AlwaysNil\}\]`
func fn4() error                  { notAStub(); return nil }    // want fn4:`nilness: \[\{AlwaysNil AlwaysNil\}\]`

func gen2() (out1 interface{}) { // want gen2:`nilness: \[\{NeverNil NeverNil\}\]`
	return 1
}

func gen3() (out1 interface{}) { // want gen3:`nilness: \[\{MaybeNil NeverNil\}\]`
	m := map[int]*int{}
	return m[0]
}

func gen4() (out1 int, out2 interface{}, out3 *int) { // want gen4:`nilness: \[\{[^ ]+ [^ ]+\} \{MaybeNil NeverNil\} \{[^ ]+ AlwaysNil\}\]`
	m := map[int]*int{}
	return 0, m[0], nil
}

func gen5() (out1 interface{}) { // want gen5:`nilness: \[\{MaybeNil NeverNil\}\]`
	return gen3()
}

func gen6(b bool) interface{} {
	if b {
		m := map[int]*int{}
		return m[0]
	} else {
		return nil
	}
}

func gen7() (out1 interface{}) { // want gen7:`nilness: \[\{AlwaysNil NeverNil\}\]`
	return fn1()
}

func gen8(x *int) (out1 interface{}) { // want gen8:`nilness: \[\{MaybeNil NeverNil\}\]`
	if x == nil {
		return x
	}
	return x
}

func gen9() (out1 interface{}) { // want gen9:`nilness: \[\{AlwaysNil NeverNil\}\]`
	var x *int
	return x
}

func gen10() (out1 interface{}) { // want gen10:`nilness: \[\{MaybeNil NeverNil\}\]`
	var x *int
	if x == nil {
		return x
	}
	return errors.New("")
}

func gen11() interface{} { // want gen11:`\[\{AlwaysNil MaybeNil\}\]`
	if true {
		return nil
	} else {
		return (*int)(nil)
	}
}

func gen12(b bool) (out1 interface{}) { // want gen12:`nilness: \[\{AlwaysNil NeverNil\}\]`
	var x interface{}
	if b {
		x = (*int)(nil)
	} else {
		x = (*string)(nil)
	}
	return x
}

func gen13() (out1 interface{}) { // want gen13:`nilness: \[\{AlwaysNil NeverNil\}\]`
	_, x, _ := fn2()
	return x
}

func gen14(ch chan *int) (out1 interface{}) { // want gen14:`nilness: \[\{MaybeNil NeverNil\}\]`
	return <-ch
}

func gen15() (out1 interface{}) { // want gen15:`nilness: \[\{MaybeNil NeverNil\}\]`
	t := &T{}
	return t.x
}

var g *int = new(int)

func gen16() (out1 interface{}) { // want gen16:`nilness: \[\{MaybeNilGlobal NeverNil\}\]`
	return g
}

func gen17(x interface{}) interface{} {
	if x != nil {
		return x
	}
	return x
}

func gen18() (int, error) {
	_, err := fn3()
	if err != nil {
		return 0, errors.New("yo")
	}
	return 0, err
}

func gen19() (out interface{}) { // want gen19:`nilness: \[\{AlwaysNil MaybeNil\}\]`
	if true {
		return (*int)(nil)
	}
	return
}

func gen20() (out interface{}) { // want gen20:`nilness: \[\{AlwaysNil MaybeNil\}\]`
	if true {
		return (*int)(nil)
	}
	return
}

func gen21() error { // want gen21:`nilness: \[\{AlwaysNil MaybeNil\}\]`
	if false {
		return (*exec.Error)(nil)
	}
	return fn4()
}

func gen22() interface{} {
	return gen6(false)
}

func gen23() interface{} {
	return gen24()
}

func gen24() interface{} {
	return gen23()
}

func gen25(x interface{}) (out1 interface{}) { // want gen25:`nilness: \[\{MaybeNil NeverNil\}\]`
	return x.(interface{})
}

func gen26(x interface{}) interface{} {
	v, _ := x.(interface{})
	return v
}

func gen27(x interface{}) (out1 interface{}) {
	defer recover()
	out1 = x.(interface{})
	return out1
}

type Error struct{}

func (*Error) Error() string { return "" }

func gen28() (out1 interface{}) { // want gen28:`nilness: \[\{NeverNil NeverNil\}\]`
	x := new(Error)
	var y error = x
	return y
}

func gen29() (out1 interface{}) { // want gen29:`nilness: \[\{AlwaysNil NeverNil\}\]`
	var x *Error
	var y error = x
	return y
}

func gen30() (out1, out2 interface{}) { // want gen30:`nilness: \[\{AlwaysNil NeverNil\} \{NeverNil NeverNil\}\]`
	return gen29(), gen28()
}

func gen31() (out1 interface{}) { // want gen31:`nilness: \[\{AlwaysNil NeverNil\}\]`
	a, _ := gen30()
	return a
}

func gen32() (out1 interface{}) { // want gen32:`nilness: \[\{NeverNil NeverNil\}\]`
	_, b := gen30()
	return b
}

func gen33() (out1 interface{}) { // want gen33:`nilness: \[\{NeverNil NeverNil\}\]`
	a, b := gen30()
	_ = a
	return b
}

func gen34() (out1, out2 interface{}) { // want gen34:`nilness: \[\{AlwaysNil AlwaysNil\} \{NeverNil NeverNil\}\]`
	return nil, 1
}

func gen35() (out1 interface{}) { // want gen35:`nilness: \[\{AlwaysNil AlwaysNil\}\]`
	a, _ := gen34()
	return a
}

func gen36() (out1 interface{}) { // want gen36:`nilness: \[\{NeverNil NeverNil\}\]`
	_, b := gen34()
	return b
}
