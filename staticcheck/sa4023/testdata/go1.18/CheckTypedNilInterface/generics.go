package pkg

func tpgen1[T *int,]() T {
	return (T)(nil)
}

func bar() {
	if tpgen1() == nil {
	}
}

func tpfn1[T any](x T) {
	if any(x) == nil {
		// this is entirely possible if baz is instantiated with an interface type for T. For example: baz[error](nil)
	}
}

func tpfn2[T ~int](x T) {
	if any(x) == nil { //@ diag(`this comparison is never true`)
		// this is not possible, because T only accepts concrete types
	}
}

func tpgen3[T any](x T) any {
	return any(x)
}

func tpgen4[T ~*int](x T) any {
	return any(x)
}

func tptest() {
	_ = tpgen1() == nil

	_ = tpgen3[error](nil) == nil

	// ideally we'd flag this, but the analysis is generic-insensitive at the moment.
	_ = tpgen3[*int](nil) == nil

	_ = tpgen4[*int](nil) == nil //@ diag(`never true`)
}
