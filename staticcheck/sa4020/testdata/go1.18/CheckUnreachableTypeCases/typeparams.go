package pkg

func tp1[T any](x interface{}) {
	switch x.(type) {
	case T:
	case int:
	}
}
