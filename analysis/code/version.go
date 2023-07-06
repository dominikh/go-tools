//go:build go1.21

package code

import (
	"go/ast"
	"go/types"
)

func fileGoVersion(f *ast.File) string {
	return f.GoVersion
}

func packageGoVersion(pkg *types.Package) string {
	return pkg.GoVersion()
}
