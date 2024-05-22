package pkg

type T struct {
	F string
}

func fn1() {
	var x []interface{}
	var y []int

	for _, v := range y {
		x = append(x, v)
	}

	var a, b []int
	for _, v := range a { //@ diag(`should replace loop`)
		b = append(b, v)
	}

	var a2, b2 []int
	for i := range a2 { //@ diag(`should replace loop`)
		b2 = append(b2, a2[i])
	}

	var a3, b3 []int
	for i := range a3 { //@ diag(`should replace loop`)
		v := a3[i]
		b3 = append(b3, v)
	}

	var a4 []int
	for i := range fn6() {
		a4 = append(a4, fn6()[i])
	}

	var m map[string]int
	var c []int
	for _, v := range m {
		c = append(c, v)
	}

	var t []T
	var m2 map[string][]T

	for _, tt := range t {
		m2[tt.F] = append(m2[tt.F], tt)
	}

	var out []T
	for _, tt := range t {
		out = append(m2[tt.F], tt)
	}
	_ = out
}

func fn2() {
	var v struct {
		V int
	}
	var in []int
	var out []int

	for _, v.V = range in {
		out = append(out, v.V)
	}
}

func fn3() {
	var t []T
	var out [][]T
	var m2 map[string][]T

	for _, tt := range t {
		out = append(out, m2[tt.F])
	}
}

func fn4() {
	var a, b, c []int
	for _, v := range a {
		b = append(c, v)
	}
	_ = b
}

func fn5() {
	var t []T
	var m2 map[string][]T
	var out []T
	for _, tt := range t {
		out = append(m2[tt.F], tt)
	}
	_ = out
}

func fn6() []int {
	return []int{1, 2, 3}
}

func fn7() {
	var x []int
	for _, v := range fn6() { //@ diag(`should replace loop`)
		// Purity doesn't matter here
		x = append(x, v)
	}

	for i := range fn6() {
		// Purity does matter here
		x = append(x, fn6()[i])
	}
}

func fn8() {
	// The lhs isn't allowed to refer to i
	var i int
	var x []int
	var y [][]int
	for i = range x {
		y[i] = append(y[i], x[i])
	}
	for i := range x {
		y[i] = append(y[i], x[i])
	}
}

func fn9() {
	// The lhs isn't allowed to have side effects
	bar := func() map[int][]int { return nil }
	var x []int
	for i := range x {
		bar()[0] = append(bar()[0], x[i])
	}
}
