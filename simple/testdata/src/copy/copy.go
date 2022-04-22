package pkg

func fn() {
	var b1, b2 []byte
	for i, v := range b1 { //@ diag(`should use copy`)
		b2[i] = v
	}

	for i := range b1 { //@ diag(`should use copy`)
		b2[i] = b1[i]
	}

	type T [][16]byte
	var a T
	b := make([]interface{}, len(a))
	for i := range b {
		b[i] = a[i]
	}

	var b3, b4 []*byte
	for i := range b3 { //@ diag(`should use copy`)
		b4[i] = b3[i]
	}

	var m map[int]byte
	for i, v := range b1 {
		m[i] = v
	}

}

func src() []interface{} { return nil }

func fn1() {
	// Don't flag this, the source is dynamic
	var dst []interface{}
	for i := range src() {
		dst[i] = src()[i]
	}
}

func fn2() {
	type T struct {
		b []byte
	}

	var src []byte
	var dst T
	for i, v := range src { //@ diag(`should use copy`)
		dst.b[i] = v
	}
}

func fn3() {
	var src []byte
	var dst [][]byte
	for i, v := range src { //@ diag(`should use copy`)
		dst[0][i] = v
	}
	for i, v := range src {
		// Don't flag, destination depends on loop variable
		dst[i][i] = v
	}
	for i, v := range src {
		// Don't flag, destination depends on loop variable
		dst[v][i] = v
	}
}

func fn4() {
	var b []byte
	var a1 [5]byte
	var a2 [10]byte
	var a3 [5]byte

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

func fn5() {
	var src, dst []byte
	for i := 0; i < len(src); i++ { //@ diag(`should use copy`)
		dst[i] = src[i]
	}

	len := func([]byte) int { return 0 }
	for i := 0; i < len(src); i++ {
		dst[i] = src[i]
	}
}
