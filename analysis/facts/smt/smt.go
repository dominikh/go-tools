// Package smt implements a fairly naive SMT solver for the QF_BV logic. Its capabilities are limited to what is
// required for Staticcheck, it is not a general solver.
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

// OPT use pointers for types, to avoid some interface allocations; reuse instances of types

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

var Analyzer = &analysis.Analyzer{
	Name:       "smt",
	Doc:        "SMT",
	Run:        smt,
	Requires:   []*analysis.Analyzer{buildir.Analyzer},
	ResultType: reflect.TypeOf(Result{}),
}

type Result struct {
	Predicates map[ir.Value]*Sexp
}

func (r Result) Unsatisfiable(target ir.Value) bool {
	return false
}

var (
	cTrue = Const{
		value: makeValue(Bool{}),
		Value: constant.MakeBool(true),
	}
	cFalse = Const{
		value: makeValue(Bool{}),
		Value: constant.MakeBool(false),
	}
)

func smt(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		smtfn2(fn)
	}
	return Result{}, nil
}

func assert(b bool) {
	if !b {
		panic("failed assertion")
	}
}

func smtfn2(fn *ir.Function) {
	if fn.Name() == "init" {
		// Don't waste our time analysing init functions, which may initialize huge data structures
		return
	}
	// XXX handle loops

	// In the absence of back edges, our CFG has a topological ordering, and definitions always precede uses, even for
	// phi nodes.
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

	predicates := make([]Value, len(fn.Blocks))
	doPred := func(b *ir.BasicBlock, n Value) {
		predicates[b.Index] = n
	}
	pred := func(b *ir.BasicBlock) Value {
		return predicates[b.Index]
	}

	doPred(fn.Blocks[0], cTrue)

	definitions := map[ir.ID]Value{}
	doDef := func(v ir.Value, n Value) {
		definitions[v.ID()] = n
	}

	for _, p := range fn.Params {
		doDef(p, MakeVar(fromGoType(p.Type()), uint64(p.ID())))
	}

	def := func(v ir.Value) Value {
		d := definitions[v.ID()]
		if d == nil {
			panic(fmt.Sprintf("no definition for %s", v))
		}
		return d
	}

	control := func(from, to *ir.BasicBlock) Value {
		var cond Value
		switch ctrl := from.Control().(type) {
		case *ir.If:
			if from.Succs[0] == to {
				cond = def(ctrl.Cond)
			} else {
				cond = Not(def(ctrl.Cond))
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

	ites := map[Var]*Sexp{}

	const offsetIte = 1e9
	doITE := func(cond, t, f Value) Var {
		assert(t.Type().Equal(f.Type()))
		v := MakeVar(t.Type(), uint64(offsetIte+len(ites)))

		n := And(
			Or(Not(cond), Equal(v, t)),
			Or(cond, Equal(v, f)))
		ites[v] = n

		return v
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
				doDef(v, MakeConst(fromGoType(v.Type()), v.Value))

			case *ir.Sigma:
				doDef(v, def(v.X))

			case *ir.Phi:
				var ite Value = def(v.Edges[len(v.Edges)-1])

				for i, e := range v.Edges[:len(v.Edges)-1] {
					// Note that using ITE like this is only correct if the same block cannot be visited twice. That is,
					// the CFG must not have any back edges.
					ite = doITE(And(pred(b.Preds[i]), control(b.Preds[i], b)), def(e), ite)
				}
				doDef(v, ite)

			case *ir.UnOp:
				var n *Sexp
				switch v.Op {
				case token.NOT:
					n = Not(def(v.X))
				case token.SUB:
					n = Op(fromGoType(v.Type()), verbBvneg, def(v.X), nil)
				default:
					panic(v.Op.String())
				}
				doDef(v, n)

			case *ir.BinOp:
				var n *Sexp
				switch v.Op {
				case token.EQL:
					n = Equal(def(v.X), def(v.Y))
				case token.NEQ:
					n = Not(Equal(def(v.X), def(v.Y)))
				case token.ADD:
					n = Op(fromGoType(v.Type()), verbBvadd, def(v.X), def(v.Y))
				case token.SUB:
					n = Op(fromGoType(v.Type()), verbBvsub, def(v.X), def(v.Y))
				case token.LSS:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvult
					} else {
						verb = verbBvslt
					}
					n = Op(Bool{}, verb, def(v.X), def(v.Y))
				case token.GTR:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvult
					} else {
						verb = verbBvslt
					}
					n = Op(Bool{}, verb, def(v.Y), def(v.X))
				case token.GEQ:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvule
					} else {
						verb = verbBvsle
					}
					n = Op(Bool{}, verb, def(v.Y), def(v.X))
				case token.LEQ:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvule
					} else {
						verb = verbBvsle
					}
					n = Op(Bool{}, verb, def(v.X), def(v.Y))
				case token.MUL:
					n = Op(fromGoType(v.Type()), verbBvmul, def(v.X), def(v.Y))
				case token.SHL:
					n = Op(fromGoType(v.Type()), verbBvshl, def(v.X), def(v.Y))
				case token.SHR:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvlshr
					} else {
						verb = verbBvashr
					}
					n = Op(fromGoType(v.Type()), verb, def(v.X), def(v.Y))
				case token.AND:
					n = Op(fromGoType(v.Type()), verbBvand, def(v.X), def(v.Y))
				case token.OR:
					n = Op(fromGoType(v.Type()), verbBvor, def(v.X), def(v.Y))
				case token.XOR:
					n = Op(fromGoType(v.Type()), verbBvxor, def(v.X), def(v.Y))
				case token.QUO:
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvudiv
					} else {
						verb = verbBvsdiv
					}
					n = Op(fromGoType(v.Type()), verb, def(v.X), def(v.Y))
				case token.REM:
					// XXX make sure Go's % has the same semantics as bvsrem
					var verb Verb
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = verbBvurem
					} else {
						verb = verbBvsrem
					}
					n = Op(fromGoType(v.Type()), verb, def(v.X), def(v.Y))
				case token.AND_NOT:
					n = Op(fromGoType(v.Type()), verbBvand, def(v.X), Op(fromGoType(v.Y.Type()), verbBvnot, def(v.Y), nil))
				default:
					panic(v.Op.String())
				}

				doDef(v, n)
			}
		}

		if b.Index == 0 {
			continue
		}

		c := make([]Value, 0, len(b.Preds))
		// XXX is this code duplicated with func control?
		for _, prev := range b.Preds {
			var cond Value
			switch ctrl := prev.Control().(type) {
			case *ir.If:
				if prev.Succs[0] == b {
					cond = def(ctrl.Cond)
				} else {
					cond = Not(def(ctrl.Cond))
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
			c = append(c, And(pred(prev), cond))
		}

		doPred(top[i], Or(c...))
	}

	if fn.Name() == "foo" {
		var c []Value

		c8 := predicates[8]
		t50 := definitions[50]
		c = append(c, c8, t50)

		for _, ite := range ites {
			// OPT only include the ITEs we need
			c = append(c, ite)
		}

		// for _, n := range assertions {
		// 	c = append(c, n)
		// }

		// c = append(c, Var{offsetVar + 50})
		// c = append(c, Var{offsetCexec + 0})
		// c = append(c, Var{offsetCexec + 2})
		// c = append(c, Var{offsetCexec + 3})

		and := And(c...)

		for i := 0; i < 10; i++ {
			propagateEqualities(and)
			simplify(and, fn)
		}

		fmt.Println(and)
	}
}

func isZero(v Value) bool {
	k, ok := v.(Const)
	if !ok {
		return false
	}
	if _, ok = k.Type().(BitVector); !ok {
		return false
	}
	n, exact := constant.Uint64Val(k.Value)
	return n == 0 && exact
}

func substitute(n Value, equalities map[Var]Value) Value {
	if n, ok := n.(Var); ok {
		s, ok := equalities[n]
		if ok {
			return s
		} else {
			return n
		}
	}
	if n, ok := n.(*Sexp); ok {
		// OPT ideally we won't copy the sexp if it doesn't reference any variables in equalities
		cp := &Sexp{
			value: n.value,
			Verb:  n.Verb,
			In:    make([]Value, len(n.In)),
		}
		for i, in := range n.In {
			cp.In[i] = substitute(in, equalities)
		}
		return cp
	}
	return n
}

func propagateEqualities(sexp *Sexp) {
	equalities := map[Var]Value{}
	if sexp.Verb == verbAnd {
		for _, in := range sexp.In {
			if in, ok := in.(*Sexp); ok && in.Verb == verbEqual && len(in.In) == 2 {
				if lhs, ok := in.In[0].(Var); ok {
					if _, ok := equalities[lhs]; !ok {
						equalities[lhs] = in.In[1]
					}
				}
			}
		}
	}

	if len(equalities) == 0 {
		return
	}

	for i, in := range sexp.In {
		sexp.In[i] = substitute(in, equalities)
	}

	// XXX propagate in nested ands
}

func simplify(sexp *Sexp, fn *ir.Function) {
	// XXX our code is so horribly inefficient

	for i, in := range sexp.In {
		if in, ok := in.(*Sexp); ok {
			simplify(in, fn)
			if in.Verb == verbIdentity {
				sexp.In[i] = in.In[0]
			}
		}
	}

	// Constant propagation
	if len(sexp.In) == 2 {
		if x, ok := sexp.In[0].(Const); ok {
			if y, ok := sexp.In[1].(Const); ok {
				_ = x // XXX
				_ = y // XXX
				switch sexp.Verb {
				case verbBvadd:
					// XXX bitwidth, signedness, actually do the math
				case verbBvult, verbBvslt, verbBvule, verbBvsle:
					// XXX do the comparison
				}
			}
		}
	}

	switch sexp.Verb {
	case verbAnd:
		newIn := make([]Value, 0, len(sexp.In))
		for _, in := range sexp.In {
			if in == cFalse {
				*sexp = Identity(cFalse)
				return
			} else if in == cTrue {
				// skip
			} else if nested, ok := in.(*Sexp); ok && nested.Verb == verbAnd {
				newIn = append(newIn, nested.In...)
			} else {
				newIn = append(newIn, in)
			}
		}
		sexp.In = newIn

		if len(sexp.In) == 0 {
			*sexp = Identity(cTrue)
		} else if len(sexp.In) == 1 {
			*sexp = Identity(sexp.In[0])
		} else {
			nodes := map[Value]struct{}{}
			for _, in := range sexp.In {
				nodes[in] = struct{}{}
			}
			for _, in := range sexp.In {
				if in, ok := in.(*Sexp); ok && in.Verb == verbNot {
					if _, ok := nodes[in.In[0]]; ok {
						// (and ...) contains both x and (not x), which is a tautology
						*sexp = Identity(cFalse)
						return
					}
				}
			}
		}

	case verbOr:
		newIn := make([]Value, 0, len(sexp.In))
		for _, in := range sexp.In {
			if in == cTrue {
				*sexp = Identity(cTrue)
				return
			} else if in == cFalse {
				// skip
			} else if nested, ok := in.(*Sexp); ok && nested.Verb == verbOr {
				newIn = append(newIn, nested.In...)
			} else {
				newIn = append(newIn, in)
			}
		}
		sexp.In = newIn

		if len(sexp.In) == 0 {
			*sexp = Identity(cFalse)
		} else if len(sexp.In) == 1 {
			*sexp = Identity(sexp.In[0])
		} else {
			nodes := map[Value]struct{}{}
			for _, in := range sexp.In {
				nodes[in] = struct{}{}
			}
			for _, in := range sexp.In {
				if in, ok := in.(*Sexp); ok && in.Verb == verbNot {
					if _, ok := nodes[in.In[0]]; ok {
						// (or ...) contains both x and (not x), which is a tautology
						*sexp = Identity(cTrue)
						return
					}
				}
			}
		}

	case verbBvadd:
		newIn := make([]Value, 0, len(sexp.In))
		for _, in := range sexp.In {
			if isZero(in) {
				// a + 0 == 0, skip this input
			} else if inSexp, ok := in.(*Sexp); ok && inSexp.Verb == verbBvadd {
				// flatten nested bvadd
				newIn = append(newIn, inSexp.In...)
			} else {
				newIn = append(newIn, in)
			}
		}
		sexp.In = newIn

		switch len(sexp.In) {
		case 0:
			// XXX we need to use a zero bitvector of the right size
			*sexp = Identity(MakeConst(sexp.typ, constant.MakeUint64(0)))
		case 1:
			*sexp = Identity(sexp.In[0])
		}

	case verbBvsub:
		// TODO remove '0'
		*sexp = Identity(Op(sexp.Type(), verbBvadd, sexp.In[0], Op(sexp.In[1].Type(), verbBvneg, sexp.In[1], nil)))

	case verbEqual:
		if sexp.In[0] == sexp.In[1] {
			*sexp = Identity(cTrue)
		} else if x, ok := sexp.In[0].(Const); ok {
			if y, ok := sexp.In[1].(Const); ok {
				*sexp = Identity(MakeConst(Bool{}, constant.MakeBool(constant.Compare(x.Value, token.EQL, y.Value))))
			}
		} else if bvadd, ok := sexp.In[0].(*Sexp); ok && bvadd.Verb == verbBvadd {
			// XXX check len(sexp.In)
			if out, ok := sexp.In[1].(Const); ok {
				if x, ok := bvadd.In[0].(Const); ok {
					if k, ok := bvadd.In[1].(Const); ok {
						outi, _ := constant.Uint64Val(out.Value)
						ki, _ := constant.Uint64Val(k.Value)

						// XXX bitwidth, signedness; right now we assume 8 bit
						*sexp = Identity(Equal(x, MakeConst(x.typ, constant.MakeUint64(uint64(uint8(outi)-uint8(ki))))))
					}
				}
			}
		}

	case verbBvult, verbBvslt:
		// TODO no value is bvult 0

		if sexp.In[0] == sexp.In[1] {
			// no value is less than itself
			*sexp = Identity(cFalse)
		}

	case verbBvule, verbBvsle:
		// TODO bvule and bvsle can be expressed in terms of bvult/bvslt and equality

		if sexp.In[0] == sexp.In[1] {
			// every value is equal to itself
			*sexp = Identity(cTrue)
		}

		var neg Verb
		if sexp.Verb == verbBvule {
			neg = verbBvult
		} else {
			neg = verbBvslt
		}
		*sexp = Identity(Not(Op(sexp.typ, neg, sexp.In[1], sexp.In[0])))

	case verbNot:
		if sexp.In[0] == cTrue {
			*sexp = Identity(cFalse)
		} else if sexp.In[0] == cFalse {
			*sexp = Identity(cTrue)
		} else if in, ok := sexp.In[0].(*Sexp); ok {
			switch in.Verb {
			case verbNot:
				*sexp = Identity(in)
			}
		}
	}

	switch sexp.Verb {
	case verbBvadd, verbAnd, verbOr:
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
