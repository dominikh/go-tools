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
	Predicates map[ir.Value]*Sexp
}

func (r Result) Unsatisfiable(target ir.Value) bool {
	return false
}

/*
func (r Result) Unsatisfiable(target ir.Value) bool {
	if !weCanDoThis(target) {
		return false
	}
	// XXX figure out a better API. We will want to synthesize our own queries.

	p, ok := r.Predicates[target]
	if !ok {
		return false
	}

	var buf bytes.Buffer
	buf.WriteString(`
	  (set-option :produce-models false)
	  (set-option :timeout 100)`)

	// XXX handle and fix loops
	var dfs func(c Component)
	seen := map[ir.Value]struct{}{}
	seenConsts := map[ir.Value]struct{}{}
	dfs = func(c Component) {
		switch c := c.(type) {
		case SMTConstant:
		case SMTValue:
			c2, ok := r.Predicates[c.Value]
			if ok {
				// dfs(c2)
				_ = c2
			} else {
				// XXX modifying r.predicates is no bueno for concurrency
				r.Predicates[c.Value] = SMTConstant{Value: constant.MakeBool(true)}
			}
		case Ref:
			if _, ok := seen[c.Value]; ok {
				return
			}
			seen[c.Value] = struct{}{}
			if _, ok := r.Predicates[c.Value]; !ok {
				// XXX modifying r.predicates is no bueno for concurrency
				r.Predicates[c.Value] = SMTConstant{Value: constant.MakeBool(true)}
			} else {
				dfs(r.Predicates[c.Value])
			}
			if _, ok := seenConsts[c.Value]; !ok {
				fmt.Fprintf(&buf, "(declare-const %s %s)\n", c.Value.Name(), constType(c.Value))
				seenConsts[c.Value] = struct{}{}
			}
			fmt.Fprintf(&buf, "(define-fun r%s () Bool %s)\n", c.Value.Name(), r.Predicates[c.Value])

		case And:
			for _, c2 := range c {
				dfs(c2)
			}
		case Or:
			for _, c2 := range c {
				dfs(c2)
			}
		case BinaryExpression:
			if c.Op != token.EQL && c.Op != token.ASSIGN {
				dfs(c.X)
			}
			dfs(c.Y)
		}
	}

	dfs(p)
	fmt.Fprintf(&buf, "(declare-const %s %s)\n", target.Name(), constType(target))
	fmt.Fprintf(&buf, "(define-fun r%s () Bool %s)\n", target.Name(), p)
	fmt.Fprintf(&buf, "(assert r%s)\n(assert %s)\n", target.Name(), target.Name())
	fmt.Fprintf(&buf, "(check-sat)")

	// XXX don't write to buf, write directly to z3 process
	// XXX obviously stop relying on external processes eventually

	fmt.Println(buf.String())

	cmd := exec.Command("z3", "-in")
	cmd.Stdin = &buf
	b, err := cmd.CombinedOutput()
	_ = err // XXX handle error

	// XXX properly verify the output. sat or unsat or unknown, anything else is unexpected

	// log.Println(string(b))

	return string(b) == "unsat\n"
}


func smt(pass *analysis.Pass) (interface{}, error) {
	// XXX we really can't use this until we have a way to differentiate literals from named consts. we're finding
	// impossible conditions that are debugging consts…

	// XXX detect and handle loops


	predicates := map[ir.Value]Component{}

	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		for _, b := range fn.Blocks {
		instrLoop:
			for _, instr := range b.Instrs {
				if v, ok := instr.(ir.Value); ok {
					if !weCanDoThis(v) {
						continue
					}
					// OPT reuse slice
					for _, rand := range v.Operands(nil) {
						if !weCanDoThis(*rand) {
							continue instrLoop
						}
					}
				} else {
					continue
				}
				switch instr := instr.(type) {
				case *ir.Const:
					predicates[instr] = BinaryExpression{SMTValue{instr}, token.EQL, SMTConstant{instr.Value, instr.Type()}}
				case *ir.Sigma:
					ctrl, ok := instr.From.Control().(*ir.If)
					if ok {
						// XXX support other controls

						if cond, ok := ctrl.Cond.(*ir.BinOp); ok {
							// XXX support other conditions

							if !weCanDoThis(cond.X) || !weCanDoThis(cond.Y) {
								continue
							}

							var c And
							if b == instr.From.Succs[0] {
								// true branch
								c = append(c,
									BinaryExpression{SMTValue{cond.X}, cond.Op, SMTValue{cond.Y}},
									Ref{cond.X},
									Ref{cond.Y})
							} else {
								// else branch
								c = append(c,
									BinaryExpression{SMTValue{cond.X}, negate(cond.Op), SMTValue{cond.Y}},
									Ref{cond.X},
									Ref{cond.Y})
							}

							c = append(c,
								BinaryExpression{SMTValue{instr}, token.EQL, SMTValue{instr.X}},
								Ref{instr.X})
							predicates[instr] = c
						}
					}

				case *ir.BinOp:
					predicates[instr] = And{
						BinaryExpression{SMTValue{instr}, token.EQL, BinaryExpression{SMTValue{instr.X}, instr.Op, SMTValue{instr.Y}}},
						Ref{instr.X},
						Ref{instr.Y}}

				case *ir.Phi:
					var c Or
					for _, edge := range instr.Edges {
						and := And{
							BinaryExpression{SMTValue{instr}, token.EQL, SMTValue{edge}},
							Ref{edge}}
						c = append(c, and)
					}
					predicates[instr] = c
				}
			}
		}
	}

	return Result{predicates}, nil
}

func flattenAnd(and And, into And) And {
	for _, c := range and {
		switch c := c.(type) {
		case And:
			into = flattenAnd(c, into)
		default:
			into = append(into, c)
		}
	}
	return into
}

func flattenOr(or Or, into Or) Or {
	for _, c := range or {
		switch c := c.(type) {
		case Or:
			into = flattenOr(c, into)
		default:
			into = append(into, c)
		}
	}
	return into
}

*/

func smt(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		smtfn2(fn)
	}
	return Result{}, nil
}

func smtfn2(fn *ir.Function) {
	bl := builder{
		sexps: map[sexpKey]*Sexp{},
	}

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

	definitions := map[ir.Value]*Sexp{}
	cexecs := map[int]*Sexp{}
	cexecs[0] = Const(constant.MakeBool(true))

	doVar := func(v ir.Value) *Sexp {
		return Var(v)

		if n, ok := definitions[v]; ok {
			return n
		} else {
			return Var(v)
		}
	}

	control := func(from, to *ir.BasicBlock) *Sexp {
		var cond *Sexp
		switch ctrl := from.Control().(type) {
		case *ir.If:
			if from.Succs[0] == to {
				cond = doVar(ctrl.Cond)
			} else {
				cond = Not(doVar(ctrl.Cond))
			}
		case *ir.Jump:
			cond = Const(constant.MakeBool(true))
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
				definitions[v] = Const(v.Value)

			case *ir.Sigma:
				definitions[v] = doVar(v.X)

			case *ir.Phi:
				var ite *Sexp = doVar(v.Edges[len(v.Edges)-1])

				for i, e := range v.Edges[:len(v.Edges)-1] {
					ite = ITE(bl.And(cexecs[b.Preds[i].Index], control(b.Preds[i], b)), doVar(e), ite)
				}
				definitions[v] = ite

			case *ir.BinOp:
				var n *Sexp
				switch v.Op {
				case token.EQL:
					n = Equal(doVar(v.X), doVar(v.Y))
				case token.NEQ:
					n = Not(Equal(doVar(v.X), doVar(v.Y)))
				case token.ADD:
					n = Op("bvadd", doVar(v.X), doVar(v.Y))
				case token.LSS:
					var verb string
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = "bvult"
					} else {
						verb = "bvslt"
					}
					n = Op(verb, doVar(v.X), doVar(v.Y))
				case token.GTR:
					var verb string
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = "bvult"
					} else {
						verb = "bvslt"
					}
					n = Op(verb, doVar(v.Y), doVar(v.X))
				default:
					panic("XXX")
				}

				definitions[v] = n
			}
		}

		if b.Index == 0 {
			continue
		}

		c := make([]*Sexp, 0, len(b.Preds))
		for _, pred := range b.Preds {
			var cond *Sexp
			switch ctrl := pred.Control().(type) {
			case *ir.If:
				if pred.Succs[0] == b {
					cond = doVar(ctrl.Cond)
				} else {
					cond = Not(doVar(ctrl.Cond))
				}
			case *ir.Jump:
				cond = Const(constant.MakeBool(true))
			default:
				panic(fmt.Sprintf("%T", ctrl))
			}
			c = append(c, bl.And(cexecs[pred.Index], cond))
		}
		cexecs[top[i].Index] = bl.Or(c...)
	}

	and := &Sexp{Verb: "and"}
	for v, n := range definitions {
		and.In = append(and.In, Equal(Var(v), n))
		// // XXX proper fixpoint loop
		// for i := 0; i < 100; i++ {
		// 	simplify(n)
		// }
		// fmt.Printf("%s <- %s\n", v.Name(), n)
	}
	for v, n := range cexecs {
		and.In = append(and.In, Equal(Raw(fmt.Sprintf("cexec%d", v)), n))
	}
	// for i, n := range cexecs {
	// 	// XXX proper fixpoint loop
	// 	for i := 0; i < 100; i++ {
	// 		simplify(n)
	// 	}
	// 	fmt.Printf("cexec%d <- %s\n", i, n)
	// }
	for i := 0; i < 100; i++ {
		simplify(and)
	}
	fmt.Println(and)
}

// XXX better name
func deepFuckingClone(sexp *Sexp) *Sexp {
	out := &Sexp{}
	*out = *sexp
	out.In = make([]*Sexp, len(sexp.In))
	for i, in := range sexp.In {
		out.In[i] = deepFuckingClone(in)
	}
	return out
}

func simplify(sexp *Sexp) {
	// XXX our code is so horribly inefficient

	if sexp.Verb == "and" {
		// propagate equalities

		type propagation struct {
			to   *Sexp
			skip *Sexp
		}

		var propagate func(sexp *Sexp, propagations map[ir.Value]propagation)
		propagate = func(sexp *Sexp, propagations map[ir.Value]propagation) {
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

		propagations := map[ir.Value]propagation{}
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

	switch sexp.Verb {
	case "and":
		switch len(sexp.In) {
		case 0:
			*sexp = *Const(constant.MakeBool(true))
		case 1:
			*sexp = *sexp.In[0]
		default:
			newIn := make([]*Sexp, 0, len(sexp.In))
		inLoop:
			for _, in := range sexp.In {
				switch in.Verb {
				case "const":
					switch in.Constant {
					case constant.MakeBool(true):
						// skip
					case constant.MakeBool(false):
						// the entire (and ...) is false
						*sexp = *in
						break inLoop
					default:
						newIn = append(newIn, in)
					}

				case "and":
					// flatten nested (and ...)
					newIn = append(newIn, in.In...)

				default:
					newIn = append(newIn, in)
				}
			}
			sexp.In = newIn

			// XXX don't use O(n²) algorithm for finding negations
		inLoop4:
			for _, in1 := range sexp.In {
				if in1.Verb == "not" {
					for _, in2 := range sexp.In {
						if in1.In[0].Equal(in2) {
							// (and ...) containing both x and !x is trivially false
							*sexp = *Const(constant.MakeBool(false))
							break inLoop4
						}
					}
				}
			}
		}

	case "or":
		switch len(sexp.In) {
		case 0:
			*sexp = *Const(constant.MakeBool(false))
		case 1:
			*sexp = *sexp.In[0]
		default:
			newIn := make([]*Sexp, 0, len(sexp.In))
		inLoop2:
			for _, in := range sexp.In {
				switch in.Verb {
				case "const":
					switch in.Constant {
					case constant.MakeBool(false):
						// skip
					case constant.MakeBool(true):
						// the entire (or ...) is true
						*sexp = *in
						break inLoop2
					default:
						newIn = append(newIn, in)
					}

				case "or":
					// flatten nested (or ...)
					newIn = append(newIn, in.In...)

				default:
					newIn = append(newIn, in)
				}
			}
			sexp.In = newIn

			// XXX don't use O(n²) algorithm for finding negations
		inLoop3:
			for _, in1 := range sexp.In {
				if in1.Verb == "not" {
					for _, in2 := range sexp.In {
						if in1.In[0].Equal(in2) {
							// (or ...) containing both x and !x is trivially true
							*sexp = *Const(constant.MakeBool(true))
							break inLoop3
						}
					}
				}
			}
		}

	case "ite":
		if sexp.In[1].Equal(sexp.In[2]) {
			*sexp = *sexp.In[1]
		} else if sexp.In[0].Verb == "const" {
			if constant.BoolVal(sexp.In[0].Constant) {
				*sexp = *sexp.In[1]
			} else {
				*sexp = *sexp.In[2]
			}
		}

	case "bvadd":
		if sexp.In[0].Verb == "const" && sexp.In[1].Verb == "const" && len(sexp.In) == 2 {
			// XXX bitwidth, signedness
			x, _ := constant.Uint64Val(sexp.In[0].Constant)
			y, _ := constant.Uint64Val(sexp.In[1].Constant)
			*sexp = *Const(constant.MakeUint64(uint64(uint8(x) + uint8(y))))
		}

	case "=":
		if len(sexp.In) == 2 && sexp.In[0].Equal(sexp.In[1]) {
			*sexp = *Const(constant.MakeBool(true))
		}

		if len(sexp.In) == 2 {
			if bvadd := sexp.In[0]; bvadd.Verb == "bvadd" && len(bvadd.In) == 2 {
				if out := sexp.In[1]; out.Verb == "const" {
					if x := bvadd.In[0]; x.Verb != "const" {
						if k := bvadd.In[1]; k.Verb == "const" {
							_ = out
							_ = x
							_ = k

							outi, _ := constant.Uint64Val(out.Constant)
							ki, _ := constant.Uint64Val(k.Constant)

							sexp.In[0] = x
							// XXX bitwidth, signedness
							sexp.In[1] = Const(constant.MakeUint64(uint64(uint8(outi) - uint8(ki))))
						}
					}
				}
			}
		}

	case "bvult", "bvslt":
		if sexp.In[0].Equal(sexp.In[1]) {
			log.Println(sexp, "is impossible")
			// no value is less than itself
			*sexp = *Const(constant.MakeBool(false))
		}
	}

	switch sexp.Verb {
	case "bvadd", "and", "or":
		sort.Slice(sexp.In, func(i, j int) bool {
			// sexp > var > const
			a := sexp.In[i]
			b := sexp.In[j]

			switch a.Verb {
			case "const":
				if b.Verb != "const" {
					return false
				}

				switch a.Constant.Kind() {
				case constant.Bool:
					ta := constant.BoolVal(a.Constant)
					tb := constant.BoolVal(b.Constant)
					return !ta && tb
				case constant.Int:
					return constant.Compare(a.Constant, token.LSS, b.Constant)
				default:
					panic(fmt.Sprintf("unexpected kind %s", a.Constant.Kind()))
				}
			case "var":
				if b.Verb == "const" {
					return true
				}
				if b.Verb != "var" {
					return false
				}
				return a.Value.ID() < b.Value.ID()
			default:
				return len(a.In) < len(b.In)
			}
		})
	}
}

// func smtfn(fn *ir.Function) {
// 	bl := builder{
// 		vars:       map[ir.Value]Node{},
// 		predicates: map[ir.Value]Node{},
// 	}

// 	for _, b := range fn.DomPreorder() {
// 	instrLoop:
// 		for _, instr := range b.Instrs {
// 			v, ok := instr.(ir.Value)
// 			if !ok {
// 				continue
// 			}

// 			if !weCanDoThis(v) {
// 				continue
// 			}
// 			// OPT reuse slice
// 			for _, rand := range v.Operands(nil) {
// 				if !weCanDoThis(*rand) {
// 					continue instrLoop
// 				}
// 			}

// 			switch v := v.(type) {
// 			case *ir.Const:
// 				bl.value(v, Const{v.Value})

// 			case *ir.Sigma:
// 				// XXX track path predicates

// 				bl.value(v, Var{v.X})

// 				ctrl, ok := v.From.Control().(*ir.If)
// 				if ok {
// 					// XXX support other controls

// 					cond := ctrl.Cond
// 					if b == v.From.Succs[0] {
// 						// true branch
// 						bl.predicate(v, Var{cond})
// 					} else {
// 						// false branch
// 						bl.predicate(v, Not(Var{cond}))
// 					}
// 				}

// 			case *ir.Phi:
// 				args := make([]Node, len(v.Edges))
// 				for i, e := range v.Edges {
// 					args[i] = Var{e}
// 				}
// 				bl.value(v, Or(args...))

// 			case *ir.BinOp:
// 				var n Node
// 				switch v.Op {
// 				case token.EQL:
// 					n = Equal(Var{v.Y}, Var{v.X})
// 				case token.ADD:
// 					n = Op("+", Var{v.X}, Var{v.Y})
// 				case token.LSS:
// 					var verb string
// 					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
// 						verb = "bvult"
// 					} else {
// 						verb = "bvslt"
// 					}
// 					n = Op(verb, Var{v.X}, Var{v.Y})
// 				case token.GTR:
// 					var verb string
// 					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
// 						verb = "bvule"
// 					} else {
// 						verb = "bvsle"
// 					}
// 					n = Op(verb, Var{v.Y}, Var{v.X})
// 				default:
// 					panic("XXX")
// 				}
// 				bl.value(v, n)
// 			}
// 		}
// 	}

// 	getPredicate := func(v ir.Value) Node {
// 		if pred, ok := bl.predicates[v]; ok {
// 			return pred
// 		} else {
// 			return Const{constant.MakeBool(true)}
// 		}
// 	}

// 	seen := map[ir.Value]struct{}{}
// 	var dfs func(v ir.Value)
// 	dfs = func(v ir.Value) {
// 		// XXX handle loops
// 		if _, ok := seen[v]; ok {
// 			return
// 		}
// 		seen[v] = struct{}{}

// 		def := bl.vars[v]
// 		switch def := def.(type) {
// 		case Const:
// 			bl.predicate(v, Const{constant.MakeBool(true)})

// 		case Var:
// 			dfs(def.Value)

// 			bl.predicate(v, And(getPredicate(v), getPredicate(def.Value)))
// 		case Sexp:
// 			preds := make([]Node, len(def.In))
// 			for i, in := range def.In {
// 				dfs(in.(Var).Value)
// 				preds[i] = getPredicate(in.(Var).Value)
// 			}

// 			if _, ok := v.(*ir.Phi); ok {
// 				bl.predicate(v, Or(preds...))
// 			} else {
// 				bl.predicate(v, And(getPredicate(v), And(preds...)))
// 			}
// 		}
// 	}

// 	for v := range bl.vars {
// 		dfs(v)
// 	}

// 	for v, n := range bl.vars {
// 		log.Printf("%s <- %s | %s", v.Name(), n, bl.predicates[v])
// 	}

// 	// for v, p := range bl.predicates {
// 	// 	if v.Name() == "t50" {
// 	// 		fmt.Println(And(v, p))
// 	// 		fmt.Println()
// 	// 		fmt.Println(bl.simplify(And(v, p)))
// 	// 	}
// 	// }
// }

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
