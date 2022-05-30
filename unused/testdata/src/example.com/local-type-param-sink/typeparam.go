package tparamsource

// https://staticcheck.io/issues/1282

import "reflect"

func TypeOfType[T any]() reflect.Type { //@ used("TypeOfType", true), used("T", true)
	var t *T //@ used("t", true)
	return reflect.TypeOf(t).Elem()
}
