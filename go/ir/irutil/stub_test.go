package irutil_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"honnef.co/go/tools/go/ir/irutil"
)

func TestIsTrivialRepeatedCallsToSameHelper(t *testing.T) {
	const src = `package p
	func leaf() int { return 1 }
	func helper() int { return leaf() }
	func caller() int { helper(); return helper() }
	`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := types.NewPackage("p", "p")
	irpkg, _, err := irutil.BuildPackage(&types.Config{Importer: importer.Default()}, fset, pkg, []*ast.File{f}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !irutil.IsTrivial(irpkg.Func("caller")) {
		t.Fatal("caller of the same trivial helper twice was not considered trivial")
	}
}
