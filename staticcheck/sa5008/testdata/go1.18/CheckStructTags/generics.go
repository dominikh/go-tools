package pkg

type S1[T any] struct {
	// flag, 'any' is too permissive
	F T `json:",string"` //@ diag(`the JSON string option`)
}

type S2[T int | string] struct {
	// don't flag, all types in T are okay
	F T `json:",string"`
}

type S3[T int | complex128] struct {
	// flag, can't use ,string on complex128
	F T `json:",string"` //@ diag(`the JSON string option`)
}

type S4[T int | string] struct {
	// don't flag, pointers to stringable types are also stringable
	F *T `json:",string"`
}

type S5[T ~int | ~string, PT ~int | ~string | ~*T] struct {
	// don't flag, pointers to stringable types are also stringable
	F PT `json:",string"`
}

type S6[T int | complex128] struct {
	// flag, pointers to non-stringable types aren't stringable, either
	F *T `json:",string"` //@ diag(`the JSON string option`)
}

type S7[T int | complex128, PT *T] struct {
	// flag, pointers to non-stringable types aren't stringable, either
	F PT `json:",string"` //@ diag(`the JSON string option`)
}

type S8[T int, PT *T | complex128] struct {
	// do flag, variation of S7
	F PT `json:",string"` //@ diag(`the JSON string option`)
}

type S9[T int | *bool, PT *T | float64, PPT *PT | string] struct {
	// do flag, multiple levels of pointers aren't allowed
	F PPT `json:",string"` //@ diag(`the JSON string option`)
}

type S10[T1 *T2, T2 *T1] struct {
	// do flag, don't get stuck in an infinite loop
	F T1 `json:",string"` //@ diag(`the JSON string option`)
}

type S11[E ~int | ~complex128, T ~*E] struct {
	F T `json:",string"` //@ diag(`the JSON string option`)
}
