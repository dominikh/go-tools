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

func doBenchmark2(s string, b testing.TB) {
	strings.Replace("", "", "", 1)
}

func BenchmarkBar(b *testing.B) {
	doBenchmark("", b)
}

func BenchmarkBar2(b *testing.B) {
	doBenchmark2("", b)
}

func BenchmarkParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		strings.Replace("", "", "", 1)
	})
}

func TestWeDontPanic(t *testing.T) {
	// we're testing that calling foo doesn't panic. we're not
	// interested in the return value.
	foo(0, 0)
}
