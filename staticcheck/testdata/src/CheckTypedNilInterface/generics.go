//go:build go1.18

package pkg

func foo[T *int,]() T {
	return (T)(nil)
}

func bar() {
	if foo() == nil {
	}
}
