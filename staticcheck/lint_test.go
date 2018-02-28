package staticcheck

import (
	"testing"

	"github.com/cabify/go-tools/lint"
	"github.com/cabify/go-tools/lint/lintutil"
	"github.com/cabify/go-tools/lint/testutil"
)

func TestAll(t *testing.T) {
	c := NewChecker()
	testutil.TestAll(t, c, "")
}

func BenchmarkStdlib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := NewChecker()
		_, err := lintutil.Lint([]lint.Checker{c}, []string{"std"}, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNetHttp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c := NewChecker()
		_, err := lintutil.Lint([]lint.Checker{c}, []string{"net/http"}, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
