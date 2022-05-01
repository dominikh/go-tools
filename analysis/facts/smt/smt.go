package smt

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"log"
	"reflect"

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
		smtfn(fn)
	}
	return Result{}, nil
}

func smtfn(fn *ir.Function) {
	bl := builder{
		vars:       map[ir.Value]Node{},
		predicates: map[ir.Value]Node{},
	}

	for _, b := range fn.DomPreorder() {
	instrLoop:
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
					continue instrLoop
				}
			}

			switch v := v.(type) {
			case *ir.Const:
				bl.value(v, Const{v.Value})

			case *ir.Sigma:
				// XXX track path predicates

				bl.value(v, Var{v.X})

				ctrl, ok := v.From.Control().(*ir.If)
				if ok {
					// XXX support other controls

					cond := ctrl.Cond
					if b == v.From.Succs[0] {
						// true branch
						bl.predicate(v, Var{cond})
					} else {
						// false branch
						bl.predicate(v, Not(Var{cond}))
					}
				}

			case *ir.Phi:
				args := make([]Node, len(v.Edges))
				for i, e := range v.Edges {
					args[i] = Var{e}
				}
				bl.value(v, Or(args...))

			case *ir.BinOp:
				var n Node
				switch v.Op {
				case token.EQL:
					n = Equal(Var{v.Y}, Var{v.X})
				case token.ADD:
					n = Op("+", Var{v.X}, Var{v.Y})
				case token.LSS:
					var verb string
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = "bvult"
					} else {
						verb = "bvslt"
					}
					n = Op(verb, Var{v.X}, Var{v.Y})
				case token.GTR:
					var verb string
					if (v.X.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
						verb = "bvule"
					} else {
						verb = "bvsle"
					}
					n = Op(verb, Var{v.Y}, Var{v.X})
				default:
					panic("XXX")
				}
				bl.value(v, n)
			}
		}
	}

	getPredicate := func(v ir.Value) Node {
		if pred, ok := bl.predicates[v]; ok {
			return pred
		} else {
			return Const{constant.MakeBool(true)}
		}
	}

	seen := map[ir.Value]struct{}{}
	var dfs func(v ir.Value)
	dfs = func(v ir.Value) {
		// XXX handle loops
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}

		def := bl.vars[v]
		switch def := def.(type) {
		case Const:
			bl.predicate(v, Const{constant.MakeBool(true)})

		case Var:
			dfs(def.Value)

			bl.predicate(v, And(getPredicate(v), getPredicate(def.Value)))
		case Sexp:
			preds := make([]Node, len(def.In))
			for i, in := range def.In {
				dfs(in.(Var).Value)
				preds[i] = getPredicate(in.(Var).Value)
			}

			if _, ok := v.(*ir.Phi); ok {
				bl.predicate(v, Or(preds...))
			} else {
				bl.predicate(v, And(getPredicate(v), And(preds...)))
			}
		}
	}

	for v := range bl.vars {
		dfs(v)
	}

	for v, n := range bl.vars {
		log.Printf("%s <- %s | %s", v.Name(), n, bl.predicates[v])
	}

	// for v, p := range bl.predicates {
	// 	if v.Name() == "t50" {
	// 		fmt.Println(And(v, p))
	// 		fmt.Println()
	// 		fmt.Println(bl.simplify(And(v, p)))
	// 	}
	// }
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
