package loader

import (
	"go/build"
	"testing"
)

func BenchmarkImportBuildPackageTree(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lprog := NewProgram(&build.Default)
		if _, err := lprog.importBuildPackageTree("net/http", ".", nil); err != nil {
			b.Fatal("unexpected error", err)
		}
	}
}

func BenchmarkImport(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lprog := NewProgram(&build.Default)
		if _, _, err := lprog.Import("net/http", "."); err != nil {
			b.Fatal("unexpected error", err)
		}
	}
}
