package pkg

import "regexp"

func fn() {
	var r *regexp.Regexp
	_ = r.FindAll(nil, 0) //@ diag(`calling a FindAll method with n == 0 will return no results`)
}

func fn2() {
	regexp.MustCompile("foo(").FindAll(nil, 0) //@ diag(`calling a FindAll`)
}
