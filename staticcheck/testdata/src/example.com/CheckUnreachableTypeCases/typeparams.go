//go:build go1.18

package pkg

func tp1[T any](x interface{}) {
	switch x.(type) {
	case T:
	case int:
	}
}
