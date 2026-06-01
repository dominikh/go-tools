package pkg

import (
	"errors"
	"unsafe"
)

type (
	T    struct{ f *int }
	T2   T
	Doer interface{ Do() }
)

func fn1() *T {
	if true {
		return nil
	}
	return &T{}
}

func fn2() *T { // want fn2:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return &T{}
}

func fn3() *T { // want fn3:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return new(T)
}

func fn4() *T { // want fn4:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return fn3()
}

func fn5() *T {
	return fn1()
}

func fn6() *T2 { // want fn6:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return (*T2)(fn4())
}

func fn7() interface{} { // want fn7:`nilness: \[\{AlwaysNil AlwaysNil\}\]`
	return nil
}

func fn8() interface{} { // want fn8:`nilness: \[\{NeverNil NeverNil\}\]`
	return 1
}

func fn9() []int { // want fn9:`nilness: \[\{[^ ]+ NeverNil\}\]`
	x := []int{}
	y := x[:1]
	return y
}

func fn10(x []int) []int { // want fn10:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return x[:1]
}

func fn11(x *T) *T {
	return x
}

func fn12(x *T) *int {
	return x.f
}

func fn13() *int { // want fn13:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return new(int)
}

func fn14() []int { // want fn14:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return make([]int, 0)
}

func fn15() []int { // want fn15:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return []int{}
}

func fn16() []int { // want fn16:`nilness: \[\{[^ ]+ AlwaysNil\}\]`
	return nil
}

func fn17() error {
	if true {
		return errors.New("")
	}
	return nil
}

func fn18() (err error) { // want fn18:`nilness: \[\{MaybeNil NeverNil\}\]`
	for {
		if err = fn17(); err != nil {
			return
		}
	}
}

var x *int

func fn19() *int { // want fn19:`nilness: \[\{[^ ]+ MaybeNilGlobal\}\]`
	return x
}

func fn20() *int {
	if true {
		return x
	}
	return nil
}

func fn27[T ~struct{ F int }]() T {
	return T{0}
}

func fn28[T [8]int]() T {
	return T{}
}

func fn29[T []int]() T { // want fn29:`nilness: \[\{[^ ]+ NeverNil\}\]`
	return T{}
}

func fn30() *int { // want fn30:`nilness: \[\{[^ ]+ AlwaysNil\}\]`
	var m map[int]*int
	return m[0]
}

func fn31() (err error) { // want fn31:`nilness: \[\{AlwaysNil AlwaysNil\}\]`
	for {
		if err = fn17(); err == nil {
			return
		}
	}
}

func fn32(x *int) *int { // want fn32:`nilness: \[\{[^ ]+ NeverNil\}\]`
	_ = *x
	return x
}

func fn33(a, b, c []int, d int) (x, y, z []int) { // want fn33:`nilness: \[\{[^ ]+ MaybeNil\} \{[^ ]+ NeverNil\} \{[^}]+ MaybeNil\}\]`
	x = a[:0]
	y = b[:1]
	z = c[:d]
	return x, y, z
}

func fn34() Doer {
	// Lacking type analysis, r is {MaybeNil MaybeNil}
	var x any = new(int)
	r, _ := x.(Doer)
	return r
}

func fn40(x []int) []int { // want fn40:`nilness: \[\{[^ ]+ NeverNil\}\]`
	if x != nil {
		// x won't become nil
		return append(x)
	}
	return make([]int, 0)
}

func fn41(x []int) []int {
	if x == nil {
		// Don't propagate x's AlwaysNil
		return append(x, 1)
	}
	return nil
}

// We don't currently have the capabilities to compute more precise information
// for the following functions. The comments in these functions indicates
// what we would like to see one day.

func fn35(x any) Doer { // would want fn35:`nilness: \[\{MaybeNil NeverNil\}\]`
	v, ok := x.(Doer)
	if ok {
		return v
	} else {
		return Doer(nil)
	}
}

func fn36(x any) Doer { // would want fn36:`nilness: \[\{AlwaysNil AlwaysNil\}\]`
	v, ok := x.(Doer)
	if ok {
		return nil
	} else {
		return v
	}
}

func fn37(x any) Doer { // would want fn37:`nilness: \[\{AlwaysNil AlwaysNil\}\]`
	v, ok := x.(Doer)
	if ok == false {
		return v
	} else {
		return v
	}
}

func fn38(x any) Doer { // would want fn38:`nilness: \[\{AlwaysNil AlwaysNil\}\]`
	v, ok := x.(Doer)
	if !ok {
		return v
	} else {
		return v
	}
}

func fn39(x []int) []int { // would want fn39:`nilness: \[\{[^ ]+ NeverNil\}\]
	return append(x, 1)
}

func fn42(x unsafe.Pointer) unsafe.Pointer { // would want fn42:`nilness: \[\{_ NeverNil\]\}
	if x == nil {
		x = unsafe.Add(x, 1)
	}
	return x
}
