package pkg

import "sort"

func fn() {
	var a []string
	var b []float64
	var c sort.StringSlice

	a = sort.StringSlice(a)  //@ diag(re`sort\.StringSlice is a type.+consider using sort\.Strings instead`)
	b = sort.Float64Slice(b) //@ diag(re`sort\.Float64Slice is a type.+consider using sort\.Float64s instead`)
	c = sort.StringSlice(c)
}
