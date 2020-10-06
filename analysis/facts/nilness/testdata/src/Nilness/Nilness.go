package pkg

import "errors"

type T struct{ f *int }
type T2 T

func fn1() *T {
	if true {
		return nil
	}
	return &T{}
}

func fn2() *T { // want fn2:`never returns nil: 00000001`
	return &T{}
}

func fn3() *T { // want fn3:`never returns nil: 00000001`
	return new(T)
}

func fn4() *T { // want fn4:`never returns nil: 00000001`
	return fn3()
}

func fn5() *T {
	return fn1()
}

func fn6() *T2 { // want fn6:`never returns nil: 00000001`
	return (*T2)(fn4())
}

func fn7() interface{} {
	return nil
}

func fn8() interface{} { // want fn8:`never returns nil: 00000001`
	return 1
}

func fn9() []int { // want fn9:`never returns nil: 00000001`
	x := []int{}
	y := x[:1]
	return y
}

func fn10(x []int) []int {
	return x[:1]
}

func fn11(x *T) *T {
	return x
}

func fn12(x *T) *int {
	return x.f
}

func fn13() *int { // want fn13:`never returns nil: 00000001`
	return new(int)
}

func fn14() []int { // want fn14:`never returns nil: 00000001`
	return make([]int, 0)
}

func fn15() []int { // want fn15:`never returns nil: 00000001`
	return []int{}
}

func fn16() []int {
	return nil
}

func fn17() error {
	if true {
		return errors.New("")
	}
	return nil
}

func fn18() (err error) { // want fn18:`never returns nil: 00000001`
	for {
		if err = fn17(); err != nil {
			return
		}
	}
}
