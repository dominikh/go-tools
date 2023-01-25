// Package pkg ...
package pkg

var (
	a = ""  //@ diag(`Unicode control character U+0007`)
	b = "" //@ diag(`Unicode control characters`)
	c = "Test	test"
	d = `T
est`
	e = `Zero​Width` //@ diag(`Unicode format character U+200B`)
	f = "\u200b"
	g = "👩🏽‍🔬" //@ diag(`Unicode control character U+0007`)
	h = "👩🏽‍🔬​" //@ diag(`Unicode format and control characters`)
)
