package pkg

// Deprecated: use another function.
func DeprecatedFn() {}

type SD struct {
	// Deprecated: external don't use me
	D string
}

type SN struct {
	// Not deprecated, but named the same
	D string
}
