package pkg

import "reflect"

type wkt interface { //@ used("wkt", true)
	XXX_WellKnownType() string //@ used("XXX_WellKnownType", true)
}

var typeOfWkt = reflect.TypeOf((*wkt)(nil)).Elem() //@ used("typeOfWkt", true)

func Fn() { //@ used("Fn", true)
	_ = typeOfWkt
}

type t *int //@ used("t", true)

var _ t //@ used("_", true)
