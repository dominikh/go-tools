package pkg

import stringsalias "strings"

func fn() {
	const (
		s1 = "foo"
		s2 = "bar"
	)

	if stringsalias.ToLower(s1) == stringsalias.ToLower(s2) { //@ diag(`should use strings.EqualFold instead`)
		panic("")
	}

	if stringsalias.ToUpper(s1) == stringsalias.ToUpper(s2) { //@ diag(`should use strings.EqualFold instead`)
		panic("")
	}

	if stringsalias.ToLower(s1) != stringsalias.ToLower(s2) { //@ diag(`should use !strings.EqualFold instead`)
		panic("")
	}

	switch stringsalias.ToLower(s1) == stringsalias.ToLower(s2) { //@ diag(`should use strings.EqualFold instead`)
	case true, false:
		panic("")
	}

	if stringsalias.ToLower(s1) == stringsalias.ToLower(s2) || s1+s2 == s2+s1 { //@ diag(`should use strings.EqualFold instead`)
		panic("")
	}

	if stringsalias.ToLower(s1) > stringsalias.ToLower(s2) {
		panic("")
	}

	if stringsalias.ToLower(s1) < stringsalias.ToLower(s2) {
		panic("")
	}

	if stringsalias.ToLower(s1) == stringsalias.ToUpper(s2) {
		panic("")
	}
}
