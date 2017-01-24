package pkg

func fn(a int, s []int, f float64) {
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
	if f == f {
	}
	if f != f {
	}
}
