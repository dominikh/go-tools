package pkg

import "reflect"

type wkt interface { //@ used(true)
	XXX_WellKnownType() string //@ used(true)
}

var typeOfWkt = reflect.TypeOf((*wkt)(nil)).Elem() //@ used(true)

func Fn() { //@ used(true)
	_ = typeOfWkt
}

type t *int //@ used(true)

var _ t
