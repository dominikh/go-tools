//go:build go1.18

package pkg

// Make sure we don't crash upon seeing a MultiConvert instruction.
func generic1[T []byte | string](s T) T {
	switch v := any(s).(type) {
	case string:
		return T(v)
	case []byte:
		return T(v)
	default:
		return s
	}
}

// Make sure we don't emit a fact for a function whose return type isn't pointer-like.
func generic2[T [4]byte | string](s T) T {
	switch v := any(s).(type) {
	case string:
		return T([]byte(v))
	case [4]byte:
		return T(v[:])
	default:
		return s
	}
}

// Make sure we detect that the return value cannot be nil. It is either a string, a
// non-nil slice we got passed, or a non-nil slice we allocate. Note that we don't
// understand that the switch's non-default branches are exhaustive over the type set and
// for the fact to be computed, we have to return something non-nil from the unreachable
// default branch.
func generic3[T []byte | string](s T) T { // want generic3:`nilness: \[\{[^ ]+ NeverNil\}\]`
	switch v := any(s).(type) {
	case string:
		return T(v)
	case []byte:
		if v == nil {
			return T([]byte{})
		} else {
			return T(v)
		}
	default:
		return T([]byte{})
	}
}

// Make sure we don't crash on ordering comparisons involving the zero value of
// a type parameter, which we represent as an ir.Const with a nil Value, just
// like an actual nil.
func generic4[T ~int | ~string](s string, parse func(string) (T, error)) (T, error) {
	n, err := parse(s)
	var zero T
	if err != nil || n <= zero {
		return zero, err
	}
	return n, nil
}
