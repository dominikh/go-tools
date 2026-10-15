package irutil

import (
	"go/token"
	"go/types"

	"honnef.co/go/tools/go/ir"
)

// Terminates reports whether fn is supposed to return, that is if it
// has at least one theoretic path that returns from the function.
// Explicit panics do not count as terminating.
func Terminates(fn *ir.Function) bool {
	if fn.Blocks == nil {
		// assuming that a function terminates is the conservative
		// choice
		return true
	}

	for _, block := range fn.Blocks {
		if _, ok := block.Control().(*ir.Return); ok {
			if len(block.Preds) == 0 {
				return true
			}
			for _, pred := range block.Preds {
				switch ctrl := pred.Control().(type) {
				case *ir.Panic:
					// explicit panics do not count as terminating
				case *ir.If:
					// Check if we got here by receiving from a closed
					// time.Tick channel – this cannot happen at
					// runtime and thus doesn't constitute termination
					cond := ctrl.Cond
					negated := false
					if unop, ok := cond.(*ir.UnOp); ok && unop.Op == token.NOT {
						cond = unop.X
						negated = true
					}
					ex, ok := cond.(*ir.Extract)
					if !ok || ex.Index != 1 {
						return true
					}
					recv, ok := ex.Tuple.(*ir.Recv)
					if !ok {
						return true
					}
					call, ok := recv.Chan.(*ir.Call)
					if !ok {
						return true
					}
					fn, ok := call.Common().Value.(*ir.Function)
					if !ok {
						return true
					}
					fn2, ok := fn.Object().(*types.Func)
					if !ok {
						return true
					}
					if fn2.FullName() != "time.Tick" {
						return true
					}
					// A receive from time.Tick always reports ok == true. The
					// return is reachable only on the true edge, accounting for
					// an optional logical negation of the condition.
					found := false
					for i, succ := range pred.Succs {
						if succ == block {
							found = true
							if (i == 0) != negated {
								return true
							}
							break
						}
					}
					if !found {
						return true
					}
				default:
					// we've reached the exit block
					return true
				}
			}
		}
	}
	return false
}
