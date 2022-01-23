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
	"go/constant"
	"go/token"
	"go/types"
	"math/big"

	"honnef.co/go/tools/go/ir"
)

type Interval struct {
	// A nil bound either indicates infinity or ⊥, depending on the value of Unknown.
	Lower, Upper *big.Int
	Unknown      bool
}

func (ival *Interval) String() string {
	if ival.Unknown {
		return "[⊥, ⊥]"
	} else {
		l := "-∞"
		u := "∞"
		if ival.Lower != nil {
			l = ival.Lower.String()
		}
		if ival.Upper != nil {
			u = ival.Upper.String()
		}
		return fmt.Sprintf("[%s, %s]", l, u)
	}
}

func (ival *Interval) IsUndefined() bool {
	return ival.Unknown
}

// TODO: we should be able to represent both intersections using a single type
type Intersection interface {
	fmt.Stringer
}

type BasicIntersection struct {
	Interval Interval
}

func (isec BasicIntersection) String() string {
	return isec.Interval.String()
}

// A SymbolicIntersection represents an intersection with an interval bounded by a comparison instruction between two
// variables. For example, for 'if a < b', in the true branch 'a' will be bounded by [min, b - 1], where 'min' is the
// smallest value representable by 'a'.
type SymbolicIntersection struct {
	Op    token.Token
	Value ir.Value
}

func (isec SymbolicIntersection) String() string {
	l := "-∞"
	u := "∞"
	name := isec.Value.Name()
	switch isec.Op {
	case token.LSS:
		u = name + "-1"
	case token.GTR:
		l = name + "+1"
	case token.LEQ:
		u = name
	case token.GEQ:
		l = name
	case token.EQL:
		l = name
		u = name
	default:
		panic(fmt.Sprintf("unhandled token %s", isec.Op))
	}
	return fmt.Sprintf("[%s, %s]", l, u)
}

type Operation interface {
	Interval() Intersection
	Eval() Interval
}

func infinity() Interval {
	// XXX should unsigned integers be [-inf, inf] or [0, inf]?
	return Interval{}
}

// flipToken flips a binary operator. For example, '>' becomes '<'.
func flipToken(tok token.Token) token.Token {
	switch tok {
	case token.LSS:
		return token.GTR
	case token.GTR:
		return token.LSS
	case token.LEQ:
		return token.GEQ
	case token.GEQ:
		return token.LEQ
	case token.EQL:
		return token.EQL
	case token.NEQ:
		return token.NEQ
	default:
		panic(fmt.Sprintf("unhandled token %v", tok))
	}
}

// negateToken negates a binary operator. For example, '>' becomes '<='.
func negateToken(tok token.Token) token.Token {
	switch tok {
	case token.LSS:
		return token.GEQ
	case token.GTR:
		return token.LEQ
	case token.LEQ:
		return token.GTR
	case token.GEQ:
		return token.LSS
	case token.EQL:
		return token.NEQ
	case token.NEQ:
		return token.EQL
	default:
		panic(fmt.Sprintf("unhandled token %s", tok))
	}
}

var one = big.NewInt(1)

type constraintGraph struct {
	// OPT: if we wrap ir.Value in a struct with some fields, then we only need one map, which reduces the number of
	// lookups and the memory usage.

	// Map sigma nodes to their intersections. In SSI form, only sigma nodes will have intersections. Only conditionals
	// cause intersections, and conditionals always cause the creation of sigma nodes for all relevant values.
	intersections map[*ir.Sigma]Intersection
	// The subset of fn's instructions that make up our constraint graph.
	nodes map[ir.Value]struct{}
	// Map instructions to computed intervals
	intervals map[ir.Value]Interval
}

func XXX(fn *ir.Function) {
	cg := constraintGraph{
		intersections: map[*ir.Sigma]Intersection{},
		nodes:         map[ir.Value]struct{}{},
		intervals:     map[ir.Value]Interval{},
	}

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

			cg.nodes[v] = struct{}{}

			if v, ok := v.(*ir.Sigma); ok {
				cg.intersections[v] = BasicIntersection{Interval: infinity()}
				// OPT: we repeat many checks for all sigmas in a basic block, even though most information is the same
				// for all sigmas, and the remaining information only matters for at most two sigmas. It might make
				// sense to either cache most of the computation, or to map from control instruction to sigma node, not
				// the other way around.
				switch ctrl := v.From.Control().(type) {
				case *ir.If:
					cond, ok := ctrl.Cond.(*ir.BinOp)
					if ok {
						lc, _ := cond.X.(*ir.Const)
						rc, _ := cond.Y.(*ir.Const)
						if lc != nil && rc != nil {
							// Comparing two constants, which isn't interesting to us
						} else if (lc != nil && rc == nil) || (lc == nil && rc != nil) {
							// Comparing a variable with a constant
							var variable ir.Value
							var k *ir.Const
							var op token.Token
							if lc != nil {
								// constant on the left side
								variable = cond.Y
								k = lc
								op = flipToken(cond.Op)
							} else {
								// constant on the right side
								variable = cond.X
								k = rc
								op = cond.Op
							}
							if variable == v.X {
								if v.From.Succs[1] == b {
									// We're in the else branch
									op = negateToken(op)
								}
								val := big.NewInt(0)
								switch k := constant.Val(k.Value).(type) {
								case int64:
									val = big.NewInt(k)
								case *big.Int:
									val.Set(k)
								default:
									panic(fmt.Sprintf("unexpected type %T", k))
								}
								switch op {
								case token.LSS:
									// [-∞, k-1]
									cg.intersections[v] = BasicIntersection{Interval{Upper: val.Sub(val, one)}}
								case token.GTR:
									// [k+1, ∞]
									cg.intersections[v] = BasicIntersection{Interval{Lower: val.Add(val, one)}}
								case token.LEQ:
									// [-∞, k]
									cg.intersections[v] = BasicIntersection{Interval{Upper: val}}
								case token.GEQ:
									// [k, ∞]
									cg.intersections[v] = BasicIntersection{Interval{Lower: val}}
								case token.NEQ:
									// We cannot represent this constraint
									// [-∞, ∞]
									cg.intersections[v] = BasicIntersection{infinity()}
								case token.EQL:
									// [k, k]
									cg.intersections[v] = BasicIntersection{Interval{Lower: val, Upper: val}}
								default:
									panic(fmt.Sprintf("unhandled token %s", op))
								}
							} else {
								// Conditional isn't about this variable
							}
						} else if lc == nil && rc == nil {
							// Comparing two variables
							if cond.X == cond.Y {
								// Comparing variable with itself, nothing to do"
							} else if cond.X != v.X && cond.Y != v.X {
								// Conditional isn't about this variable
							} else {
								var variable ir.Value
								var op token.Token
								if cond.X == v.X {
									// Our variable on the left side
									variable = cond.Y
									op = cond.Op
								} else {
									// Our variable on the right side
									variable = cond.X
									op = flipToken(cond.Op)
								}

								if v.From.Succs[1] == b {
									// We're in the else branch
									op = negateToken(op)
								}

								switch op {
								case token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL:
									cg.intersections[v] = SymbolicIntersection{op, variable}
								case token.NEQ:
									// We cannot represent this constraint
									// [-∞, ∞]
									cg.intersections[v] = BasicIntersection{infinity()}
								default:
									panic(fmt.Sprintf("unhandled token %s", op))
								}
							}
						} else {
							panic("unreachable")
						}
					} else {
						// We don't know how to derive new information from the branch condition.
					}
				// case *ir.ConstantSwitch:
				default:
					panic(fmt.Sprintf("unhandled control %T", ctrl))
				}
			}
		}
	}

	printConstraintGraph(cg.nodes, cg.intersections)
}

func printConstraintGraph(nodes map[ir.Value]struct{}, intersections map[*ir.Sigma]Intersection) {
	for v := range nodes {
		switch v := v.(type) {
		case *ir.Sigma:
			fmt.Printf("%s = %s ∩ %s\n", v.Name(), v.X.Name(), intersections[v])
		case *ir.Const:
			fmt.Printf("%s = %s\n", v.Name(), v.Value)
		default:
			fmt.Printf("%s = %s\n", v.Name(), v)
		}
	}
}

func eval(instr ir.Instruction) bool {
	return false
}
