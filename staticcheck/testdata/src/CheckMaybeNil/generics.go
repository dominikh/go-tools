//go:build go1.18

package pkg

func tpfn1[T []int](x T) {
	// don't flag, T is a slice
	_ = x[0]
	if x == nil {
		return
	}
	println()
}

func tpfn2[T *int,](x T) {
	_ = *x //@ diag(`possible nil pointer dereference`)
	if x == nil {
		return
	}
	println()
}
