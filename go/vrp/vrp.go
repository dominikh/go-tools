// Package vrp implements value range analysis on Go programs in SSI form.
//
// We implement the algorithm shown in the paper "Speed And Precision in Range Analysis" by Campos et al. Further resources discussing this algorithm are:
// - Scalable and precise range analysis on the interval lattice by Rodrigues
// - A Fast and Low Overhead Technique to Secure Programs Against Integer Overflows by Rodrigues et al
// - https://github.com/vhscampos/range-analysis
// - https://www.youtube.com/watch?v=Vj-TI4Yjt10
//
// TODO: document use of jump-set widening, possible use of rounds of abstract interpretation, what our lattice looks like, ...
package vrp

import (
	"fmt"
	"go/token"
	"go/types"
	"math/big"

	"honnef.co/go/tools/go/ir"
)

type Value struct{}

type Interval struct {
	Lower, Upper *big.Int
}

func (ival *Interval) IsUndefined() bool {
	if ival.Lower == nil && ival.Upper != nil || ival.Lower != nil && ival.Upper == nil {
		panic("inconsistent interval")
	}
	return ival.Lower == nil && ival.Upper == nil
}

// TODO: we should be able to represent both intersections using a single type
type Intersection interface{}

type BasicIntersection struct {
	Interval Interval
}

// A SymbolicIntersection represents an intersection with an interval bounded by a comparison instruction between two
// variables. For example, for 'if a < b', in the true branch 'a' will be bounded by [min, b - 1], where 'min' is the
// smallest value representable by 'a'.
type SymbolicIntersection struct {
	Op    token.Token
	Value *Value
}

type Operation interface {
	Interval() Intersection
	Eval() Interval
}

func XXX(fn *ir.Function) {
	// OPT: if we wrap ir.Value in a struct with some fields, then we only need one map, which reduces the number of
	// lookups and the memory usage.

	// Map sigma nodes to their intersections. In SSI form, only sigma nodes will have intersections. Only conditionals
	// cause intersections, and conditionals always lead to sigma nodes for all relevant values.
	intersections := map[*ir.Sigma]Intersection{}
	// The subset of fn's instructions that make up our constraint graph.
	nodes := map[ir.Value]struct{}{}
	// Map instructions to computed intervals
	intervals := map[ir.Value]Interval{}

	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			v, ok := instr.(ir.Value)
			if !ok {
				continue
			}
			basic, ok := v.Type().Underlying().(*types.Basic)
			if !ok {
				continue
			}
			if (basic.Info() & types.IsInteger) == 0 {
				continue
			}

			switch instr := instr.(type) {
			case *ir.Const:
				// XXX
			case *ir.Sigma:
				// OPT: we repeat many checks for all sigmas in a basic block, even though most information is the
				// same for all sigmas, and the remaining information only matters for at most two sigmas.
				switch ctrl := instr.From.Control().(type) {
				case *ir.If:
					cond, ok := ctrl.Cond.(*ir.BinOp)
					if ok {
						lc, _ := cond.X.(*ir.Const)
						rc, _ := cond.Y.(*ir.Const)
						if lc != nil && rc != nil {
							// Comparing two constants, which isn't interesting to us
							// XXX propagate
						} else if (lc != nil && rc == nil) || (lc == nil && rc != nil) {
							// Comparing a variable with a constant
							// XXX
							var variable ir.Value
							if lc != nil {
								// constant on the left side
								variable = cond.Y
							} else {
								// constant on the right side
								variable = cond.X
							}
							if variable != instr.X {
								// Conditional isn't about this variable
								// XXX propagate
							} else {
							}
						} else if lc == nil && rc == nil {
							// Comparing two variables
							// XXX
							if cond.X != instr.X && cond.Y != instr.X {
								// Conditional isn't about this variable
								// XXX propagate
							} else {
							}
						}
						if instr.From.Succs[0] == b {
							// We're in the true branch
							// XXX
						} else {
							// We're in the false branch
							// XXX
						}
					} else {
						// We don't know how to derive new information from the branch condition.
						// XXX propagate
					}
				// case *ir.ConstantSwitch:
				default:
					panic(fmt.Sprintf("unhandled control %T", ctrl))
				}
			default:
				panic(fmt.Sprintf("unhandled instruction %T", instr))
			}
		}
	}
}
