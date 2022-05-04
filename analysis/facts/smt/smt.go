package smt

// XXX canonical ordering of inputs
// XXX we can solve things like (= (bvadd Var Const) Const) directly, without going through SAT. do we need ITE for this?
// XXX figure out a better graph representation and on the fly simplifications

// TODO rewrites to apply:
// (= (bvadd v c1) c2) => (= v <computed>)
// (= (bvadd x y) x) => (= y 0)
// (= (bvadd x y) y) => (= x 0)
// (bvadd x x) => (bvshl x 1)
// (bvadd x 0) => x
// (<op> c1 c2) -> <computed>
// canonical ordering; values before consts
// (bvule <max value> x) -> (= x <max value>)
// (bvule 0 x) -> true
// (or ... x ... !x ...) -> true
// (and ... x ... !x ...) -> false
// (and (and ...)) -> flatten
// (or (or ...)) -> flatten
// (ite false a b) -> b
// (ite true a b) -> a
// (not (const k)) -> !k
// (= x true) -> x
// (= x false) -> (not x)
//
// propagate equalities, both formulas '(= x foo)' and terms 'x'

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"log"
	"reflect"
	"sort"

	"golang.org/x/tools/go/analysis"
	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/internal/passes/buildir"
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

func smt(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		smtfn2(fn)
	}
	return Result{}, nil
}

func smtfn2(fn *ir.Function) {
	// XXX handle loops

	// mapping from basic block to list of execution conditions
	// execConditions := [][]ir.Value

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

	definitions := map[ir.Value]Node{}
	cexecs := map[int]Node{}
	cexecs[0] = Const{constant.MakeBool(true)}

	control := func(from, to *ir.BasicBlock) Node {
		var cond Node
		switch ctrl := from.Control().(type) {
		case *ir.If:
			if from.Succs[0] == to {
				cond = Var{ctrl.Cond.Name()}
			} else {
				cond = Not(Var{ctrl.Cond.Name()})
			}
		case *ir.Jump:
			cond = Const{constant.MakeBool(true)}
		default:
			panic(fmt.Sprintf("%T", ctrl))
		}
		return cond
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
				definitions[v] = Const{v.Value}

			case *ir.Sigma:
				definitions[v] = Var{v.X.Name()}

			case *ir.Phi:
				var ite Node = Var{v.Edges[len(v.Edges)-1].Name()}

				for i, e := range v.Edges[:len(v.Edges)-1] {
					ite = ITE(And(Var{fmt.Sprintf("cexec%d", b.Preds[i].Index)}, control(b.Preds[i], b)), Var{e.Name()}, ite)
				}
				definitions[v] = ite

			case *ir.BinOp:
				var n Node
				switch v.Op {
				case token.EQL:
					n = Equal(Var{v.X.Name()}, Var{v.Y.Name()})
				case token.NEQ:
					n = Not(Equal(Var{v.X.Name()}, Var{v.Y.Name()}))
				case token.ADD:
					n = Op("bvadd", Var{v.X.Name()}, Var{v.Y.Name()})
				case token.LSS:
					var verb string
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = "bvult"
					} else {
						verb = "bvslt"
					}
					n = Op(verb, Var{v.X.Name()}, Var{v.Y.Name()})
				case token.GTR:
					var verb string
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = "bvult"
					} else {
						verb = "bvslt"
					}
					n = Op(verb, Var{v.Y.Name()}, Var{v.X.Name()})
				default:
					panic("XXX")
				}

				definitions[v] = n
			}
		}

		if b.Index == 0 {
			continue
		}

		c := make([]Node, 0, len(b.Preds))
		for _, pred := range b.Preds {
			var cond Node
			switch ctrl := pred.Control().(type) {
			case *ir.If:
				if pred.Succs[0] == b {
					cond = Var{ctrl.Cond.Name()}
				} else {
					cond = Not(Var{ctrl.Cond.Name()})
				}
			case *ir.Jump:
				cond = Const{constant.MakeBool(true)}
			default:
				panic(fmt.Sprintf("%T", ctrl))
			}
			c = append(c, And(Var{fmt.Sprintf("cexec%d", pred.Index)}, cond))
		}
		cexecs[top[i].Index] = Or(c...)
	}

	if fn.Name() == "foo" {
		var c []Node
		for v, n := range definitions {
			c = append(c, Equal(Var{v.Name()}, n))
			if v.Name() == "t50" {
				c = append(c, n)
			}
		}

		for i, n := range cexecs {
			c = append(c, Equal(Var{fmt.Sprintf("cexec%d", i)}, n))
		}

		c = append(c, cexecs[8])
		and := And(c...)
		for i := 0; i < 100; i++ {
			and = simplify(and)
		}
		fmt.Println(and)
	}
}

func simplify(n Node) Node {
	// XXX our code is so horribly inefficient

	/*
		if sexp.Verb == "and" {
			// propagate equalities

			type propagation struct {
				to   *Sexp
				skip *Sexp
			}

			var propagate func(sexp *Sexp, propagations map[string]propagation)
			propagate = func(sexp *Sexp, propagations map[string]propagation) {
				for i, in := range sexp.In {
					if in.Verb == "var" {
						if prop, ok := propagations[in.Value]; ok && !prop.skip.Equal(sexp) {
							sexp.In[i] = prop.to
						}
					} else {
						propagate(in, propagations)
					}
				}
			}

			propagations := map[string]propagation{}
			for _, in := range sexp.In {
				if in.Verb == "=" && len(in.In) == 2 && in.In[0].Verb == "var" {
					if _, ok := propagations[in.In[0].Value]; !ok {
						propagations[in.In[0].Value] = propagation{in.In[1], in}
					}
				}
			}

			if len(propagations) != 0 {
				*sexp = *deepFuckingClone(sexp)
				propagate(sexp, propagations)
			}
		}

		for _, in := range sexp.In {
			simplify(in)
		}
	*/

	cTrue := Const{constant.MakeBool(true)}
	cFalse := Const{constant.MakeBool(false)}
	cZero := Const{constant.MakeUint64(0)}
	if sexp, ok := n.(Sexp); ok {
		for i, in := range sexp.In {
			sexp.In[i] = simplify(in)
		}

		switch sexp.Verb {
		case "and":
			if sexp.In[0] == cFalse || sexp.In[1] == cFalse {
				return cFalse
			} else if sexp.In[0] == cTrue {
				return sexp.In[1]
			} else if sexp.In[1] == cTrue {
				log.Println("replacing", sexp, "with", sexp.In[0])
				return sexp.In[0]
			}

			// TODO find conflicting negation, recursively

		case "or":
			if sexp.In[0] == cTrue || sexp.In[1] == cTrue {
				return cTrue
			} else if sexp.In[0] == cFalse {
				return sexp.In[1]
			} else if sexp.In[1] == cFalse {
				return sexp.In[0]
			}

			// TODO find conflicting negation, recursively

		case "ite":
			if sexp.In[1].Equal(sexp.In[2]) {
				return sexp.In[1]
			} else if c, ok := sexp.In[0].(Const); ok {
				if constant.BoolVal(c.Value) {
					return sexp.In[1]
				} else {
					return sexp.In[2]
				}
			}

		case "bvadd":
			if x, ok := sexp.In[0].(Const); ok {
				if y, ok := sexp.In[1].(Const); ok {
					// XXX bitwidth, signedness
					xi, _ := constant.Uint64Val(x.Value)
					yi, _ := constant.Uint64Val(y.Value)
					return Const{constant.MakeUint64(uint64(uint8(xi) + uint8(yi)))}
				}
			}

			if sexp.In[0] == cZero {
				// (bvadd 0 x) => x
				return sexp.In[1]
			} else if sexp.In[1] == cZero {
				// (bvadd x 0) => x
				return sexp.In[0]
			}

		case "=":
			if sexp.In[0].Equal(sexp.In[1]) {
				return cTrue
			}

			if x, ok := sexp.In[0].(Const); ok {
				if y, ok := sexp.In[1].(Const); ok {
					return Const{constant.MakeBool(constant.Compare(x.Value, token.EQL, y.Value))}
				}
			}

			if bvadd, ok := sexp.In[0].(Sexp); ok && bvadd.Verb == "bvadd" {
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

		case "bvult", "bvslt":
			if sexp.In[0].Equal(sexp.In[1]) {
				// no value is less than itself
				return cFalse
			}

		case "not":
			if sexp.In[0] == cTrue {
				return cFalse
			} else if sexp.In[0] == cFalse {
				return cTrue
			}
		}

		switch sexp.Verb {
		case "bvadd", "and", "or":
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
