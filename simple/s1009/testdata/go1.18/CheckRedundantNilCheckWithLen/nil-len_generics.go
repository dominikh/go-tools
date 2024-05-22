package pkg

func fn1[T []int | *[4]int](a T) {
	if a != nil && len(a) > 0 { // don't flag, because of the pointer
	}
}

func fn2[T []int | []string | map[string]int](a T) {
	if a != nil && len(a) > 0 { //@ diag(`should omit nil check`)
	}
}
