package pkg

func tpfn[T any]() {
	var b1, b2 []T
	for i, v := range b1 { //@ diag(`should use copy`)
		b2[i] = v
	}

	for i := range b1 { //@ diag(`should use copy`)
		b2[i] = b1[i]
	}

	type T2 [][16]T
	var a T2
	b := make([]any, len(a))
	for i := range b {
		b[i] = a[i]
	}

	var b3, b4 []*T
	for i := range b3 { //@ diag(`should use copy`)
		b4[i] = b3[i]
	}

	var m map[int]T
	for i, v := range b1 {
		m[i] = v
	}

}

func tpsrc[T any]() []T { return nil }

func tpfn1() {
	// Don't flag this, the source is dynamic
	var dst []any
	for i := range tpsrc[any]() {
		dst[i] = tpsrc[any]()[i]
	}
}

func tpfn2[T any]() {
	type T2 struct {
		b []T
	}

	var src []T
	var dst T2
	for i, v := range src { //@ diag(`should use copy`)
		dst.b[i] = v
	}
}

func tpfn3[T any]() {
	var src []T
	var dst [][]T
	for i, v := range src { //@ diag(`should use copy`)
		dst[0][i] = v
	}
	for i, v := range src {
		// Don't flag, destination depends on loop variable
		dst[i][i] = v
	}
}

func tpfn4[T any]() {
	var b []T
	var a1 [5]T
	var a2 [10]T
	var a3 [5]T

	for i := range b { //@ diag(`should use copy`)
		a1[i] = b[i]
	}
	for i := range a1 { //@ diag(`should use copy`)
		b[i] = a1[i]
	}
	for i := range a1 { //@ diag(`should use copy`)
		a2[i] = a1[i]
	}
	for i := range a1 { //@ diag(`should copy arrays using assignment`)
		a3[i] = a1[i]
	}

	a1p := &a1
	a2p := &a2
	a3p := &a3
	for i := range b { //@ diag(`should use copy`)
		a1p[i] = b[i]
	}
	for i := range a1p { //@ diag(`should use copy`)
		b[i] = a1p[i]
	}
	for i := range a1p { //@ diag(`should use copy`)
		a2p[i] = a1p[i]
	}
	for i := range a1p { //@ diag(`should copy arrays using assignment`)
		a3p[i] = a1p[i]
	}

	for i := range a1 { //@ diag(`should use copy`)
		a2p[i] = a1[i]
	}
	for i := range a1 { //@ diag(`should copy arrays using assignment`)
		a3p[i] = a1[i]
	}
	for i := range a1p { //@ diag(`should use copy`)
		a2[i] = a1p[i]
	}
	for i := range a1p { //@ diag(`should copy arrays using assignment`)
		a3[i] = a1p[i]
	}
}

func tpfn5[T any]() {
	var src, dst []T
	for i := 0; i < len(src); i++ { //@ diag(`should use copy`)
		dst[i] = src[i]
	}

	len := func([]T) int { return 0 }
	for i := 0; i < len(src); i++ {
		dst[i] = src[i]
	}
}
