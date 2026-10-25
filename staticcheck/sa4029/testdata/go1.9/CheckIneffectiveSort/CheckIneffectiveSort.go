package pkg

import sorting "sort"

func fn() {
	type Strings = []string
	var d Strings

	d = sorting.StringSlice(d) //@ diag(re`sort\.StringSlice is a type.+consider using sort\.Strings instead`)
}
