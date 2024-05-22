package pkg

import "sort"

func fn() {
	type Strings = []string
	var d Strings

	d = sort.StringSlice(d) //@ diag(re`sort\.StringSlice is a type.+consider using sort\.Strings instead`)
}
