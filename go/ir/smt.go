package ir

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
)

func (And) isComponent()              {}
func (Or) isComponent()               {}
func (Ref) isComponent()              {}
func (BinaryExpression) isComponent() {}
func (SMTConstant) isComponent()      {}
func (SMTValue) isComponent()         {}

type Component interface {
	String() string
	isComponent()
}

type And []Component

func (and And) String() string {
	parts := make([]string, len(and))
	for i, c := range and {
		parts[i] = c.String()
	}
	return fmt.Sprintf("(and %s)", strings.Join(parts, " "))
}

type Or []Component

func (or Or) String() string {
	parts := make([]string, len(or))
	for i, c := range or {
		parts[i] = c.String()
	}
	return fmt.Sprintf("(or %s)", strings.Join(parts, " "))
}

type Ref struct {
	Value Value
}

func (ref Ref) String() string {
	// return fmt.Sprintf("(ref %s)", ref.Value.Name())
	return fmt.Sprintf("r%s", ref.Value.Name())
}

type BinaryExpression struct {
	X  Component
	Op token.Token
	Y  Component
}

func (expr BinaryExpression) String() string {
	op := expr.Op
	if op == token.EQL {
		op = token.ASSIGN
	}
	if op == token.NEQ {
		return fmt.Sprintf("(not (= %s %s))", expr.X, expr.Y)
	} else {
		return fmt.Sprintf("(%s %s %s)", op, expr.X, expr.Y)
	}
}

type SMTConstant struct {
	Value constant.Value
}

func (k SMTConstant) String() string {
	return k.Value.ExactString()
}

type SMTValue struct {
	Value Value
}

func (v SMTValue) String() string {
	return v.Value.Name()
}

func constType(v Value) string {
	var typ string
	// XXX handle integers correctly, i.e. use bit vectors, and use signed/unsigned shifts.
	switch v.Type().Underlying().(*types.Basic).Kind() {
	case types.Bool:
		typ = "Bool"
	case types.Int:
		typ = "Int"
	case types.Int8:
		typ = "Int"
	case types.Int16:
		typ = "Int"
	case types.Int32:
		typ = "Int"
	case types.Int64:
		typ = "Int"
	case types.Uint:
		typ = "Int"
	case types.Uint8:
		typ = "Int"
	case types.Uint16:
		typ = "Int"
	case types.Uint32:
		typ = "Int"
	case types.Uint64:
		typ = "Int"
	case types.Uintptr:
		typ = "Int"
	case types.Float32:
		panic("XXX")
	case types.Float64:
		panic("XXX")
	default:
		panic(fmt.Sprintf("unexpected type %s", v.Type()))
	}
	return typ
}

func smt(fn *Function) {
	negate := func(op token.Token) token.Token {
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
	predicates := map[Value]Component{}

	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			switch instr := instr.(type) {
			case *Const:
				predicates[instr] = BinaryExpression{SMTValue{instr}, token.EQL, SMTConstant{instr.Value}}
			case *Sigma:
				ctrl, ok := instr.From.Control().(*If)
				if ok {
					// XXX support other controls

					if cond, ok := ctrl.Cond.(*BinOp); ok {
						// XXX support other conditions

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

			case *BinOp:
				predicates[instr] = And{
					BinaryExpression{SMTValue{instr}, token.EQL, BinaryExpression{SMTValue{instr.X}, instr.Op, SMTValue{instr.Y}}},
					Ref{instr.X},
					Ref{instr.Y}}

			case *Phi:
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

	// XXX handle and fix loops
	var dfs func(c Component)
	seen := map[Value]struct{}{}
	seenConsts := map[Value]struct{}{}
	dfs = func(c Component) {
		switch c := c.(type) {
		case SMTConstant:
		case SMTValue:
			c2, ok := predicates[c.Value]
			if ok {
				// dfs(c2)
				_ = c2
			} else {
				predicates[c.Value] = SMTConstant{constant.MakeBool(true)}
			}
			// if _, ok := seenConsts[c.Value]; !ok {
			// 	fmt.Printf("(declare-const %s %s)\n", c.Value.Name(), constType(c.Value))
			// 	seenConsts[c.Value] = struct{}{}
			// }
		case Ref:
			if _, ok := seen[c.Value]; ok {
				return
			}
			seen[c.Value] = struct{}{}
			if _, ok := predicates[c.Value]; !ok {
				predicates[c.Value] = SMTConstant{constant.MakeBool(true)}
			} else {
				dfs(predicates[c.Value])
			}
			if _, ok := seenConsts[c.Value]; !ok {
				fmt.Printf("(declare-const %s %s)\n", c.Value.Name(), constType(c.Value))
				seenConsts[c.Value] = struct{}{}
			}
			fmt.Printf("(define-fun r%s () Bool %s)\n", c.Value.Name(), predicates[c.Value])

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

	// Simulate input to an API
	var target Value
	for v := range predicates {
		if v.Name() == "t29" {
			target = v
			break
		}
	}

	if target == nil {
		return
	}

	dfs(predicates[target])
	fmt.Printf("(declare-const %s %s)\n", target.Name(), constType(target))
	fmt.Printf("(define-fun r%s () Bool %s)\n", target.Name(), predicates[target])
	fmt.Printf("(assert r%s)\n(assert %s)\n", target.Name(), target.Name())

	return

	for _, c := range predicates {
		// XXX avoid repeating work
		dfs(c)
	}

	// XXX generate rt functions for variables with no constraints
	// XXX properly declare-const everything
	// XXX detect and handle loops

	// XXX use the correct types instead of Int. proper signedness, proper bitwidth.

	fmt.Println(fn)
	for v, pred := range predicates {
		fmt.Printf("(define-fun r%s () Bool %s)\n", v.Name(), pred)
	}
	fmt.Println()
}
