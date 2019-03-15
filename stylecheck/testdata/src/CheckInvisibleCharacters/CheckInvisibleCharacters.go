// Package pkg ...
package pkg

var (
	a = "" // MATCH "Unicode control character U+0007"
	b = ""
	c = "Test	test"
	d = `T
est`
	e = `Zero​Width` // MATCH "Unicode format character U+200B"
	f = "\u200b"
)

// MATCH:6 "Unicode control character U+0007"
// MATCH:6 "Unicode control character U+001A"
