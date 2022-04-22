//go:build (one || two || three || go1.1) && (three || one || two || go1.1)
// +build one two three go1.1
// +build three one two go1.1

package pkg //@ diag(`identical build constraints`)
