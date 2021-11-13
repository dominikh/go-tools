package pkg

func fn() {
	var b1, b2 []byte
	for i, v := range b1 { // want `should use copy`
		b2[i] = v
	}

	for i := range b1 { // want `should use copy`
		b2[i] = b1[i]
	}

	type T [][16]byte
	var a T
	b := make([]interface{}, len(a))
	for i := range b {
		b[i] = a[i]
	}

	var b3, b4 []*byte
	for i := range b3 { // want `should use copy`
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
	for i, v := range src { // want `should use copy`
		dst.b[i] = v
	}
}

func fn3() {
	var src []byte
	var dst [][]byte
	for i, v := range src { // want `should use copy`
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
