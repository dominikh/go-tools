package pkg

type Float float64

type Floats [5]float64
type Ints [5]int

type T1 struct {
	A float64
	B float64
}

type T2 struct {
	A float64
	B int
}

func fn(a int, s []int, f1 float64, f2 Float, fs Floats, is Ints, t1 T1, t2 T2) {
	if 0 == 0 { // want `identical expressions`
		println()
	}
	if 1 == 1 { // want `identical expressions`
		println()
	}
	if a == a { // want `identical expressions`
		println()
	}
	if a != a { // want `identical expressions`
		println()
	}
	if s[0] == s[0] { // want `identical expressions`
		println()
	}
	if 1&1 == 1 { // want `identical expressions`
		println()
	}
	if (1 + 2 + 3) == (1 + 2 + 3) { // want `identical expressions`
		println()
	}
	if f1 == f1 {
		println()
	}
	if f1 != f1 {
		println()
	}
	if f1 > f1 { // want `identical expressions`
		println()
	}
	if f2 == f2 {
		println()
	}
	if fs == fs {
		println()
	}
	if is == is { // want `identical expressions`
		println()
	}
	if t1 == t1 {
		println()
	}
	if t2 == t2 { // want `identical expressions`
		println()
	}
}
