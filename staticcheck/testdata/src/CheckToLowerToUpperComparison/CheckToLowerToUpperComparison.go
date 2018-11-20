package pkg

import "strings"

func fn() {
	const (
		s1 = "foo"
		s2 = "bar"
	)

	if strings.ToLower(s1) == strings.ToLower(s2) { // MATCH "strings.ToLower(a) == strings.ToLower(b) is better written as strings.EqualFold(a, b)"
		panic("")
	}

	if strings.ToUpper(s1) == strings.ToUpper(s2) { // MATCH "strings.ToUpper(a) == strings.ToUpper(b) is better written as strings.EqualFold(a, b)"
		panic("")
	}

	if strings.ToLower(s1) != strings.ToLower(s2) { // MATCH "strings.ToLower(a) != strings.ToLower(b) is better written as !strings.EqualFold(a, b)"
		panic("")
	}

	switch strings.ToLower(s1) == strings.ToLower(s2) { // MATCH "strings.ToLower(a) == strings.ToLower(b) is better written as strings.EqualFold(a, b)"
	case true, false:
		panic("")
	}

	if strings.ToLower(s1) == strings.ToLower(s2) || s1+s2 == s2+s1 { // MATCH "strings.ToLower(a) == strings.ToLower(b) is better written as strings.EqualFold(a, b)" {
		panic("")
	}

	if strings.ToLower(s1) > strings.ToLower(s2) {
		panic("")
	}

	if strings.ToLower(s1) < strings.ToLower(s2) {
		panic("")
	}

	if strings.ToLower(s1) == strings.ToUpper(s2) {
		panic("")
	}
}
