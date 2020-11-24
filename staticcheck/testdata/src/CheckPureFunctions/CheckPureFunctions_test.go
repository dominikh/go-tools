package pkg

import (
	"strings"
	"testing"
)

func TestFoo(t *testing.T) {
	strings.Replace("", "", "", 1) // want `only makes sense if its return values get used`
}

func BenchmarkFoo(b *testing.B) {
	strings.Replace("", "", "", 1)
}

func doBenchmark(s string, b *testing.B) {
	strings.Replace("", "", "", 1)
}

func BenchmarkBar(b *testing.B) {
	doBenchmark("", b)
}
