package ssautil

import (
	"golang.org/x/tools/go/packages"
	"honnef.co/go/tools/ssa"
)

// CreateProgram returns a new program in SSA form, given a program
// loaded from source.  An SSA package is created for each transitively
// error-free package of lprog.
//
// Code for bodies of functions is not built until Build is called
// on the result.
//
// mode controls diagnostics and checking during SSA construction.
//
func CreateProgram(pkgs []*packages.Package, mode ssa.BuilderMode) *ssa.Program {
	prog := ssa.NewProgram(pkgs[0].Fset, mode)

	for _, pkg := range pkgs {
		if !pkg.IllTyped {
			prog.CreatePackage(pkg.Types, pkg.Syntax, pkg.TypesInfo, true)
		}
	}

	return prog
}

func Reachable(from, to *ssa.BasicBlock) bool {
	if from == to {
		return true
	}
	if from.Dominates(to) {
		return true
	}

	found := false
	Walk(from, func(b *ssa.BasicBlock) bool {
		if b == to {
			found = true
			return false
		}
		return true
	})
	return found
}

func Walk(b *ssa.BasicBlock, fn func(*ssa.BasicBlock) bool) {
	seen := map[*ssa.BasicBlock]bool{}
	wl := []*ssa.BasicBlock{b}
	for len(wl) > 0 {
		b := wl[len(wl)-1]
		wl = wl[:len(wl)-1]
		if seen[b] {
			continue
		}
		seen[b] = true
		if !fn(b) {
			continue
		}
		wl = append(wl, b.Succs...)
	}
}
