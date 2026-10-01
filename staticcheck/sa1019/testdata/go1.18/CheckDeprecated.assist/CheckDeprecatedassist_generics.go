package pkg

type S[T any] struct {
	Field1 T
	// Deprecated: don't use me
	Field2 T
}

func (S[T]) Foo() {}

// Deprecated: don't use me
func (S[T]) Bar() {}

// Deprecated: don't use me
func (S[T]) Baz() {}

func (S[T]) Qux() {}

type S2[T any] struct {
	Field1 T
	// Deprecated: don't use me
	Field2 string
}

type S3[T, U any] struct {
	Field1 T
	Field2 U
	// Deprecated: don't use me
	Field3 string
}
