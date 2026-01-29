package pkg

type ErrGeneric[T any] struct {
	Value T
}

func (e ErrGeneric[T]) Error() string { return "" }

func fn() {
	// Should warn: instantiated with uncomparable type
	_ = error(ErrGeneric[[]string]{}) //@ diag(`conversion of uncomparable type`)

	// Should NOT warn: instantiated with comparable type
	_ = error(ErrGeneric[int]{})
	_ = error(ErrGeneric[string]{})

	// Should NOT warn: pointer
	_ = error(&ErrGeneric[[]string]{})
}

func fnTypeParam[T any](v T) {
	type LocalErr struct {
		Val T
	}
	// Type parameter - don't warn (can't know comparability at compile time)
	// This doesn't implement error anyway, just testing IR handling
}
