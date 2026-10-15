package pkg

import . "strings"

func fn(a, b string) {
	_ = ToLower(a) == ToLower(b) //@ diag(`should use strings.EqualFold`)
	_ = ToUpper(a) != ToUpper(b) //@ diag(`should use !strings.EqualFold`)
}
