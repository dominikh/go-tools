// Package pkg ...
package pkg

func fn(x string, y int) {
	if "" == x { // MATCH "Yoda"
	}
	if 0 == y { // MATCH "Yoda"
	}
	if 0 > y {
	}
	if "" == "" {
	}

	if "" == "" || 0 == y { // MATCH "Yoda"
	}
}
