package quickfix

import "honnef.co/go/tools/analysis/lint"

var Docs = map[string]*lint.Documentation{
	"QF1000": {
		Title: "Use byte-specific indexing function",
		Since: "Unreleased",
	},
	"QF1001": {
		Title: "Apply De Morgan's law",
		Since: "Unreleased",
	},
}
