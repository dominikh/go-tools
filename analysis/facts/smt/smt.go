// Package smt implements a fairly naive SMT solver for the QF_BV logic.
package smt

// XXX we can solve things like (= (bvadd Var Const) Const) directly, without going through SAT. do we need ITE for this?
// XXX figure out a better graph representation and on the fly simplifications

// TODO rewrites to apply:
// (= (bvadd v c1) c2) => (= v <computed>)
// (= (bvadd x y) x) => (= y 0)
// (= (bvadd x y) y) => (= x 0)
// (bvadd x x) => (bvshl x 1)
// (bvadd x 0) => x
// (<op> c1 c2) -> <computed>
// (bvule <max value> x) -> (= x <max value>)
// (bvule 0 x) -> true
//
// (and (bvslt a b) (bvslt b a)) -> false
// a < b ∧ b < c implies a < c, which helps with (and (bvslt a b) (bvslt b c) (bvslt c a)), as we can expand it to (and (bvslt a b) (bvslt b c) (bvslt a c) (bvslt c a)) and find the contradiction
// do the same for <=, not just <, and also for the unsigned versions
// do the same for 'or', resulting in true
//
// propagate equalities, both formulas '(= x foo)' and terms 'x'

/*
dee: BV-Constraint f  ̈ur “teure” Operationen wird nur hinzugef  ̈ugt
falls die Formel ohne diese Operationen erf  ̈ullbar ist
Alternative: Approximiere die “teuren” Operationen im ersten Schritt
durch uninterpretierte Funktionen
Benutze Ackermanns Reduktion um die uninterpretierten Funktionen
durch Variablen zu ersetzen
*/

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"reflect"
	"sort"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/internal/passes/buildir"
)

// XXX we're using ir.Value's IDs; make sure we don't break on globals

const (
	offsetVar   = 0
	offsetCexec = 1e9
	offsetIte   = 2e9
)

var Analyzer = &analysis.Analyzer{
	Name:       "smt",
	Doc:        "SMT",
	Run:        smt,
	Requires:   []*analysis.Analyzer{buildir.Analyzer},
	ResultType: reflect.TypeOf(Result{}),
}

type Result struct {
	Predicates map[ir.Value]Node
}

func (r Result) Unsatisfiable(target ir.Value) bool {
	return false
}

var cTrue = Const{constant.MakeBool(true)}
var cFalse = Const{constant.MakeBool(false)}
var cZero = Const{constant.MakeUint64(0)}
var cOne = Const{constant.MakeUint64(1)}

func smt(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		smtfn2(fn)
	}
	return Result{}, nil
}

func smtfn2(fn *ir.Function) {
	if fn.Name() == "init" {
		// Don't waste our time analysing init functions, which may initialize huge data structures
		return
	}
	// XXX handle loops

	seen := make([]bool, len(fn.Blocks))
	var dfs func(b *ir.BasicBlock)
	top := make([]*ir.BasicBlock, 0, len(fn.Blocks))
	dfs = func(b *ir.BasicBlock) {
		if seen[b.Index] {
			return
		}
		seen[b.Index] = true
		for _, succ := range b.Succs {
			dfs(succ)
		}
		top = append(top, b)
	}
	dfs(fn.Blocks[0])

	var assertions []Node
	assertions = append(assertions, Equal(Var{offsetCexec + 0}, cTrue))

	control := func(from, to *ir.BasicBlock) Node {
		var cond Node
		switch ctrl := from.Control().(type) {
		case *ir.If:
			if from.Succs[0] == to {
				cond = Var{offsetVar + uint64(ctrl.Cond.ID())}
			} else {
				cond = Not(Var{offsetVar + uint64(ctrl.Cond.ID())})
			}
		case *ir.Jump:
			cond = cTrue
		case *ir.Panic:
			cond = cTrue
		default:
			panic(fmt.Sprintf("%T", ctrl))
		}
		return cond
	}

	iteCnt := uint64(0)

	doITE := func(cond, t, f Node) Var {
		v := Var{offsetIte + iteCnt}
		iteCnt++

		n := And(
			Or(Not(cond), Equal(v, t)),
			Or(cond, Equal(v, f)))
		assertions = append(assertions, n)

		return v
	}

	doDef := func(v ir.Value, n Node) {
		assertions = append(assertions, Equal(Var{offsetVar + uint64(v.ID())}, n))
	}

	for i := len(top) - 1; i >= 0; i-- {
		b := top[i]

		for _, instr := range b.Instrs {
			v, ok := instr.(ir.Value)
			if !ok {
				continue
			}

			if !weCanDoThis(v) {
				continue
			}
			// OPT reuse slice
			for _, rand := range v.Operands(nil) {
				if !weCanDoThis(*rand) {
					continue
				}
			}

			switch v := v.(type) {
			case *ir.Const:
				doDef(v, Const{v.Value})

			case *ir.Sigma:
				doDef(v, Var{offsetVar + uint64(v.X.ID())})

			case *ir.Phi:
				var ite Node = Var{offsetVar + uint64(v.Edges[len(v.Edges)-1].ID())}

				for i, e := range v.Edges[:len(v.Edges)-1] {
					ite = doITE(And(Var{offsetCexec + uint64(b.Preds[i].Index)}, control(b.Preds[i], b)), Var{offsetVar + uint64(e.ID())}, ite)
				}
				doDef(v, ite)

			case *ir.UnOp:
				var n Node
				switch v.Op {
				case token.NOT:
					n = Not(Var{offsetVar + uint64(v.X.ID())})
				case token.SUB:
					n = Op(verbBvneg, Var{offsetVar + uint64(v.X.ID())}, nil)
				default:
					panic(v.Op.String())
				}
				doDef(v, n)

			case *ir.BinOp:
				var n Node
				switch v.Op {
				case token.EQL:
					n = Equal(Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.NEQ:
					n = Not(Equal(Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())}))
				case token.ADD:
					n = Op(verbBvadd, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.SUB:
					n = Op(verbBvsub, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.LSS:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvult
					} else {
						verb = verbBvslt
					}
					n = Op(verb, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.GTR:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvult
					} else {
						verb = verbBvslt
					}
					n = Op(verb, Var{offsetVar + uint64(v.Y.ID())}, Var{offsetVar + uint64(v.X.ID())})
				case token.GEQ:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvule
					} else {
						verb = verbBvsle
					}
					n = Op(verb, Var{offsetVar + uint64(v.Y.ID())}, Var{offsetVar + uint64(v.X.ID())})
				case token.LEQ:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvule
					} else {
						verb = verbBvsle
					}
					n = Op(verb, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.MUL:
					n = Op(verbBvmul, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.SHL:
					n = Op(verbBvshl, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.SHR:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvlshr
					} else {
						verb = verbBvashr
					}
					n = Op(verb, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.AND:
					n = Op(verbBvand, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.OR:
					n = Op(verbBvor, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.XOR:
					n = Op(verbBvxor, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.QUO:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvudiv
					} else {
						verb = verbBvsdiv
					}
					n = Op(verb, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.REM:
					// XXX make sure Go's % has the same semantics as bvsrem
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvurem
					} else {
						verb = verbBvsrem
					}
					n = Op(verb, Var{offsetVar + uint64(v.X.ID())}, Var{offsetVar + uint64(v.Y.ID())})
				case token.AND_NOT:
					n = Op(verbBvand, Var{offsetVar + uint64(v.X.ID())}, Op(verbBvnot, Var{offsetVar + uint64(v.Y.ID())}, nil))
				default:
					panic(v.Op.String())
				}

				doDef(v, n)
			}
		}

		if b.Index == 0 {
			continue
		}

		c := make([]Node, 0, len(b.Preds))
		// XXX is this code duplicated with func control?
		for _, pred := range b.Preds {
			var cond Node
			switch ctrl := pred.Control().(type) {
			case *ir.If:
				if pred.Succs[0] == b {
					cond = Var{offsetVar + uint64(ctrl.Cond.ID())}
				} else {
					cond = Not(Var{offsetVar + uint64(ctrl.Cond.ID())})
				}
			case *ir.Jump:
				cond = cTrue
			case *ir.ConstantSwitch:
				// XXX implement this
				cond = cTrue
			case *ir.Panic:
				cond = cTrue
			case *ir.Unreachable:
				cond = cFalse
			default:
				panic(fmt.Sprintf("%T", ctrl))
			}
			c = append(c, And(Var{offsetCexec + uint64(pred.Index)}, cond))
		}

		assertions = append(assertions, Equal(Var{offsetCexec + uint64(top[i].Index)}, Or(c...)))
	}

	if fn.Name() == "commonPrefixLenIgnoreCase" {
		f := And(Equal(Var{1}, Not(Var{1})), nil)

		// f := And(Equal(Var{999}, And(Var{999}, cTrue)), cTrue)
		fmt.Println(f)
		fmt.Println(simplify(f, nil, fn))

		// var c []Node

		// for _, n := range assertions {
		// 	fmt.Println(n)
		// 	c = append(c, n)
		// }

		// // c = append(c, Var{offsetVar + 50})
		// // c = append(c, Var{offsetCexec + 8})

		// and := And(c...)

		// for i := 0; i < 5; i++ {
		// 	and = simplify(and, nil, fn)
		// }

		// // fmt.Println(and)
	}
}

func verbToOp(verb Verb) token.Token {
	switch verb {
	case verbBvult:
		return token.LSS
	case verbBvslt:
		return token.LSS
	case verbBvule:
		return token.LEQ
	case verbBvsle:
		return token.LEQ
	default:
		// XXX
		panic(verb)
	}
}

func simplify(n Node, parent Node, fn *ir.Function) Node {
	// callers := make([]uintptr, 5000)
	// XXX := runtime.Callers(0, callers)
	// fmt.Println(strings.Repeat(" ", XXX), n)
	round := 0
	if p, ok := parent.(Sexp); !ok || p.Verb != verbAnd {
		// XXX this code is fucking broken for self-referential nodes like (= x (and x ... )) - it will blow up to gigabytes of formulae
		// XXX it is also wrong for (= x (not x))
		for {
			round++
			fmt.Println(round)
			if sexp, ok := n.(Sexp); ok && sexp.Verb == verbAnd {
				fmt.Println(n)
				equalities := map[Var]Node{}
				addEquality := func(node Node) {
					sexp, ok := node.(Sexp)
					if !ok || sexp.Verb != verbEqual {
						return
					}
					lhs, ok := sexp.In[0].(Var)
					if !ok {
						return
					}
					if _, ok := equalities[lhs]; ok {
						return
					}
					equalities[lhs] = sexp.In[1]
				}
				// Propagate equalities
				var dfs func(node Node, depth int)
				dfs = func(node Node, depth int) {
					sexp, ok := node.(Sexp)
					if !ok || sexp.Verb != verbAnd {
						return
					}

					addEquality(sexp.In[0])
					addEquality(sexp.In[1])

					dfs(sexp.In[0], depth+1)
					dfs(sexp.In[1], depth+1)
				}
				dfs(sexp, 0)

				var rename func(n Node) Node
				rename = func(n Node) Node {
					switch n := n.(type) {
					case Sexp:
						if n.Verb == verbEqual {
							if r := rename(n.In[0]); r != n.In[1] {
								n.In[0] = r
							}
							n.In[1] = rename(n.In[1])
							return n
						} else {
							for i, in := range n.In {
								n.In[i] = rename(in)
							}
							return n
						}
					case Var:
						if r, ok := equalities[n]; ok && r != n {
							return r
						} else {
							return n
						}
					case Const:
						return n
					case nil:
						return nil
					default:
						panic(fmt.Sprintf("%T", n))
					}
				}
				new := rename(n)
				if n == new {
					break
				}
				n = new
			} else {
				break
			}

			// XXX avoid O(n²). we see an And, we run this code. we later simplify the children of And, which might also be
			// And, but don't deserve to have this applied again.
		}
	}

	// XXX our code is so horribly inefficient

	hasBoth := func(root Sexp) bool {
		seen := map[Node]struct{}{}
		var dfs func(n Node)
		found := false
		dfs = func(n Node) {
			// XXX clean up this code

			seen[n] = struct{}{}
			if sexp, ok := n.(Sexp); ok {
				if sexp.Verb == verbNot {
					if _, ok := seen[sexp.In[0]]; ok {
						found = true
						return
					}
				} else {
					if _, ok := seen[Not(sexp)]; ok {
						found = true
						return
					}
				}

				if sexp.Verb == root.Verb {
					dfs(sexp.In[0])
					dfs(sexp.In[1])
				}
			} else {
				if _, ok := seen[Not(sexp)]; ok {
					found = true
					return
				}
			}
		}

		dfs(root.In[0])
		dfs(root.In[1])
		return found
	}

	if sexp, ok := n.(Sexp); ok {
		for i, in := range sexp.In {
			sexp.In[i] = simplify(in, n, fn)
		}

		// Constant propagation
		if x, ok := sexp.In[0].(Const); ok {
			if y, ok := sexp.In[1].(Const); ok {
				switch sexp.Verb {
				case verbBvadd:
					// XXX bitwidth, signedness
					xi, _ := constant.Uint64Val(x.Value)
					yi, _ := constant.Uint64Val(y.Value)
					return Const{constant.MakeUint64(uint64(uint8(xi) + uint8(yi)))}
				case verbBvult, verbBvslt, verbBvule, verbBvsle:
					op := verbToOp(sexp.Verb)
					return Const{constant.MakeBool(constant.Compare(x.Value, op, y.Value))}
				}
			}
		}

		switch sexp.Verb {
		case verbAnd:
			if sexp.In[0] == cFalse || sexp.In[1] == cFalse {
				return cFalse
			} else if sexp.In[0] == cTrue {
				return sexp.In[1]
			} else if sexp.In[1] == cTrue {
				return sexp.In[0]
			} else {
				// TODO find conflicting negation, recursively
				// Find a pair of 'x' and '(not x)'

				if parent, ok := parent.(Sexp); !ok || parent.Verb != verbAnd {
					if hasBoth(sexp) {
						return cFalse
					}
				}
			}

		case verbOr:
			if sexp.In[0] == cTrue || sexp.In[1] == cTrue {
				return cTrue
			} else if sexp.In[0] == cFalse {
				return sexp.In[1]
			} else if sexp.In[1] == cFalse {
				return sexp.In[0]
			} else {
				// Find a pair of 'x' and '(not x)'
				if hasBoth(sexp) {
					return cTrue
				}
			}

		case verbBvadd:
			if sexp.In[1] == cZero {
				// (bvadd x 0) => x
				return sexp.In[0]
			}

		case verbBvsub:
			return Op(verbBvadd, sexp.In[0], Op(verbBvneg, sexp.In[1], nil))

		// case "bvneg":
		// 	// XXX is this worth doing?
		// 	return Op(verbBvadd, Op("bvnot", sexp.In[0], nil), cOne)

		case verbEqual:
			if sexp.In[0] == sexp.In[1] {
				return cTrue
			}

			if x, ok := sexp.In[0].(Const); ok {
				if y, ok := sexp.In[1].(Const); ok {
					return Const{constant.MakeBool(constant.Compare(x.Value, token.EQL, y.Value))}
				}
			}

			if bvadd, ok := sexp.In[0].(Sexp); ok && bvadd.Verb == verbBvadd {
				if out, ok := sexp.In[1].(Const); ok {
					if x, ok := bvadd.In[0].(Const); ok {
						if k, ok := bvadd.In[1].(Const); ok {
							outi, _ := constant.Uint64Val(out.Value)
							ki, _ := constant.Uint64Val(k.Value)

							// XXX bitwidth, signedness
							return Equal(x, Const{constant.MakeUint64(uint64(uint8(outi) - uint8(ki)))})
						}
					}
				}
			}

		case verbBvult, verbBvslt:
			if sexp.In[0] == sexp.In[1] {
				// no value is less than itself
				return cFalse
			}

		case verbBvule, verbBvsle:
			// TODO bvule and bvsle can be expressed in terms of bvult/bvslt and equality

			if sexp.In[0] == sexp.In[1] {
				// every value is equal to itself
				return cTrue
			}

			var neg Verb
			if sexp.Verb == verbBvule {
				neg = verbBvult
			} else {
				neg = verbBvslt
			}
			return Not(Op(neg, sexp.In[1], sexp.In[0]))

		case verbNot:
			if sexp.In[0] == cTrue {
				return cFalse
			} else if sexp.In[0] == cFalse {
				return cTrue
			} else if in, ok := sexp.In[0].(Sexp); ok {
				switch in.Verb {
				case verbNot:
					return in
				}
			}
		}

		switch sexp.Verb {
		case verbBvadd, verbAnd, verbOr:
			// XXX there are only two arguments, we don't need sort.Slice for that

			sort.Slice(sexp.In[:], func(i, j int) bool {
				// sexp > var > const
				a := sexp.In[i]
				b := sexp.In[j]

				switch a := a.(type) {
				case Const:
					b, ok := b.(Const)
					if !ok {
						return false
					}

					switch a.Value.Kind() {
					case constant.Bool:
						ta := constant.BoolVal(a.Value)
						tb := constant.BoolVal(b.Value)
						return !ta && tb
					case constant.Int:
						return constant.Compare(a.Value, token.LSS, b.Value)
					default:
						panic(fmt.Sprintf("unexpected kind %s", a.Value.Kind()))
					}
				case Var:
					switch b := b.(type) {
					case Const:
						return true
					case Var:
						return a.Name < b.Name
					default:
						return false
					}
				default:
					return false
				}
			})
		}

		return sexp
	}

	return n
}

// XXX the name, among other things…
func weCanDoThis(v ir.Value) bool {
	if basic, ok := v.Type().Underlying().(*types.Basic); ok {
		switch basic.Kind() {
		case types.Bool:
		case types.Int:
		case types.Int8:
		case types.Int16:
		case types.Int32:
		case types.Int64:
		case types.Uint:
		case types.Uint8:
		case types.Uint16:
		case types.Uint32:
		case types.Uint64:
		case types.Uintptr:
			return false
		case types.Float32:
			return false
		case types.Float64:
			return false
		default:
			return false
		}
		return true
	} else {
		return false
	}
}

func negate(op token.Token) token.Token {
	// XXX this code exists in at least one other place -> deduplicate

	switch op {
	case token.EQL:
		return token.NEQ
	case token.NEQ:
		return token.EQL
	case token.LSS:
		return token.GEQ
	case token.GTR:
		return token.LEQ
	case token.LEQ:
		return token.GTR
	case token.GEQ:
		return token.LSS
	default:
		panic(fmt.Sprintf("unsupported token %v", op))
	}
}
