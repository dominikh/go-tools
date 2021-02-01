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
	"QF1002": {
		Title: "Convert untagged switch to tagged switch",
		Text: `An untagged switch that compares a single variable against a series of values can be replaced with a tagged switch.

Before:

	switch {
	case x == 1 || x == 2, x == 3:
		...
	case x == 4:
		...
	default:
		...
	}

After:

	switch x {
	case 1, 2, 3:
		...
	case 4:
		...
	default:
		...
	}
`,
		Since: "Unreleased",
	},
	"QF1003": {
		Title: "Convert if/else-if chain to tagged switch",
		Text: ` A series of if/else-if checks comparing the same variable against values can be replaced with a tagged switch.

Before:

	if x == 1 || x == 2 {
		...
	} else if x == 3 {
		...
	} else {
		...
	}

After:

	switch x {
	case 1, 2:
		...
	case 3:
		...
	default:
		...
	}`,
		Since: "Unreleased",
	},
	"QF1004": {
		Since: "Unreleased",
	},
	"QF1005": {
		Title: "Expand call to math.Pow",
		Text: `Some uses of math.Pow can be simplified to basic multiplication.

Before:

	math.Pow(x, 2)

After:

	x * x`,
		Since: "Unreleased",
	},

	"QF1006": {
		Title: "Lift if+break into loop condition",
		Text: `Before:

	for {
		if done {
			break
		}
		...
	}

After:

	for !done {
		...
	}`,
		Since: "Unreleased",
	},

	"QF1007": {
		Title: "Merge conditional assignment into variable declaration",
		Text: `Before:

	x := false
	if someCondition {
		x = true
	}

After:

	x := someCondition
`,
		Since: "Unreleased",
	},
}
