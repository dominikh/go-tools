//go:build go1.18

package pkg

func tpfn1[T comparable](x T) {
	if x != x {
	}
}

func tpfn2[T int | string](x T) {
	if x != x { //@ diag(`identical expressions`)
	}
}

func tpfn3[T int | float64](x T) {
	if x != x {
	}
}

func tpfn4[E int | int64, T [4]E](x T) {
	if x != x { //@ diag(`identical expressions`)
	}
}

func tpfn5[E int | float64, T [4]E](x T) {
	if x != x {
	}
}
