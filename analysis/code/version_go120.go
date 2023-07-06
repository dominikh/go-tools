//go:build !go1.21

package code

import (
	"go/ast"
	"go/types"
)

func fileGoVersion(f *ast.File) string {
	return ""
}

func packageGoVersion(pkg *types.Package) string {
	return ""
}
