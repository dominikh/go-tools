// Package pkg ...
package pkg

func fn(x string, y int) {
	if "" == x { //@ diag(`Yoda`)
	}
	if 0 == y { //@ diag(`Yoda`)
	}
	if 0 > y {
	}
	if "" == "" {
	}

	if "" == "" || 0 == y { //@ diag(`Yoda`)
	}
}
