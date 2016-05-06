package pkg

func fn() {
	if 1 == 1 { // MATCH /identical expressions/
	}
	if a == a { // MATCH /identical expressions/
	}
	if a != a { // MATCH /identical expressions/
	}
	if s[0] == s[0] { // MATCH /identical expressions/
	}
	if 1&1 == 1 { // MATCH /identical expressions/
	}
	if (1 + 2 + 3) == (1 + 2 + 3) { // MATCH /identical expressions/
	}
	var f float64
	if f == f {
	}
	if f != f {
	}
}

/*
../../testdata/lhs-rhs-identical.go:4:5: identical expressions on the left and right side of the '==' operator
*/
