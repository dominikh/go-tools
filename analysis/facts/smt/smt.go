package smt

import (
	"bytes"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"log"
	"os/exec"
	"reflect"
	"strings"

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
	Predicates map[ir.Value]Component
}

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
	  (set-option :produce-models true)
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

	log.Println(string(b))

	return string(b) == "unsat\n"
}

func (And) isComponent()              {}
func (Or) isComponent()               {}
func (Ref) isComponent()              {}
func (BinaryExpression) isComponent() {}
func (SMTConstant) isComponent()      {}
func (SMTValue) isComponent()         {}

type Component interface {
	String() string
	Equal(o Component) bool
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

func (and And) Equal(o Component) bool {
	if o, ok := o.(And); ok {
		if len(and) != len(o) {
			return false
		}
		for i := range and {
			if !and[i].Equal(o[i]) {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

type Or []Component

func (or Or) String() string {
	parts := make([]string, len(or))
	for i, c := range or {
		parts[i] = c.String()
	}
	return fmt.Sprintf("(or %s)", strings.Join(parts, " "))
}

func (or Or) Equal(o Component) bool {
	if o, ok := o.(Or); ok {
		if len(or) != len(o) {
			return false
		}
		for i := range or {
			if !or[i].Equal(o[i]) {
				return false
			}
		}
		return true
	} else {
		return false
	}
}

type Ref struct {
	Value ir.Value
}

func (ref Ref) String() string {
	// return fmt.Sprintf("(ref %s)", ref.Value.Name())
	return fmt.Sprintf("r%s", ref.Value.Name())
}

func (ref Ref) Equal(o Component) bool {
	if o, ok := o.(Ref); ok {
		return ref.Value == o.Value
	} else {
		return false
	}
}

type BinaryExpression struct {
	X  Component
	Op token.Token
	Y  Component
}

func (expr BinaryExpression) String() string {
	var op string

	// XXX this logic for figuring out the signed/unsignedness of the comparison is subpar

	var signed bool
	if x, ok := expr.X.(SMTValue); ok {
		signed = (x.Value.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) == 0
	} else if y, ok := expr.Y.(SMTValue); ok {
		signed = (y.Value.Type().Underlying().(*types.Basic).Info() & types.IsUnsigned) == 0
	}

	// XXX all of the comparison functions need to differentiate signed and unsigned

	// XXX we can use a lookup table for this
	switch expr.Op {
	case token.EQL:
		op = "="
	case token.ADD:
		op = "bvadd"
	case token.LSS:
		if signed {
			op = "bvslt"
		} else {
			op = "bvult"
		}
	case token.LEQ:
		if signed {
			op = "bvsle"
		} else {
			op = "bvule"
		}
	case token.GTR:
		if signed {
			op = "bvsgt"
		} else {
			op = "bvugt"
		}
	case token.NEQ:
		return fmt.Sprintf("(not (= %s %s))", expr.X, expr.Y)
	default:
		panic(fmt.Sprintf("unsupported token %s", expr.Op))
	}

	return fmt.Sprintf("(%s %s %s)", op, expr.X, expr.Y)
}

func (expr BinaryExpression) Equal(o Component) bool {
	if o, ok := o.(BinaryExpression); ok {
		return expr.Op == o.Op && expr.X.Equal(o.X) && expr.Y.Equal(o.Y)
	} else {
		return false
	}
}

type SMTConstant struct {
	Value constant.Value
	Type  types.Type
}

func (k SMTConstant) String() string {
	switch k.Value.Kind() {
	case constant.Bool:
		return k.Value.ExactString()
	case constant.Int:
		// XXX use correct bit widths and sign, handle imprecise
		n, _ := constant.Int64Val(k.Value)
		if k.Type != nil {
			switch k.Type.Underlying().(*types.Basic).Kind() {
			case types.Int:
				return fmt.Sprintf("(_ bv%d 64)", uint64(n))
			case types.Int8:
				return fmt.Sprintf("(_ bv%d 8)", uint64(n))
			case types.Int16:
				return fmt.Sprintf("(_ bv%d 16)", uint64(n))
			case types.Int32:
				return fmt.Sprintf("(_ bv%d 32)", uint64(n))
			case types.Int64:
				return fmt.Sprintf("(_ bv%d 64)", uint64(n))
			case types.Uint:
				return fmt.Sprintf("(_ bv%d 64)", uint64(n))
			case types.Uint8:
				return fmt.Sprintf("(_ bv%d 8)", uint64(n))
			case types.Uint16:
				return fmt.Sprintf("(_ bv%d 16)", uint64(n))
			case types.Uint32:
				return fmt.Sprintf("(_ bv%d 32)", uint64(n))
			case types.Uint64:
				return fmt.Sprintf("(_ bv%d 64)", uint64(n))
			case types.Uintptr:
				return fmt.Sprintf("(_ bv%d 64)", uint64(n))
			default:
				panic("XXX")
			}
		} else {
			// XXX this branch existing is probably a mistake
			return fmt.Sprintf("(_ bv%d 64)", uint64(n))
		}
	default:
		panic(fmt.Sprintf("unsupported kind %s", k.Value.Kind()))
	}
}

func (k SMTConstant) Equal(o Component) bool {
	if o, ok := o.(SMTConstant); ok {
		return k.Value == o.Value
	} else {
		return false
	}
}

type SMTValue struct {
	Value ir.Value
}

func (v SMTValue) String() string {
	return v.Value.Name()
}

func (v SMTValue) Equal(o Component) bool {
	if o, ok := o.(SMTValue); ok {
		return v.Value == o.Value
	} else {
		return false
	}
}

func constType(v ir.Value) string {
	var typ string
	// XXX handle integers correctly, i.e. use bit vectors, and use signed/unsigned shifts.

	// XXX don't assume that int is always 64 bits
	switch v.Type().Underlying().(*types.Basic).Kind() {
	case types.Bool:
		typ = "Bool"
	case types.Int:
		typ = "(_ BitVec 64)"
	case types.Int8:
		typ = "(_ BitVec 8)"
	case types.Int16:
		typ = "(_ BitVec 16)"
	case types.Int32:
		typ = "(_ BitVec 32)"
	case types.Int64:
		typ = "(_ BitVec 64)"
	case types.Uint:
		typ = "(_ BitVec 64)"
	case types.Uint8:
		typ = "(_ BitVec 8)"
	case types.Uint16:
		typ = "(_ BitVec 16)"
	case types.Uint32:
		typ = "(_ BitVec 32)"
	case types.Uint64:
		typ = "(_ BitVec 64)"
	case types.Uintptr:
		typ = "(_ BitVec 64)"
	case types.Float32:
		panic("XXX")
	case types.Float64:
		panic("XXX")
	default:
		panic(fmt.Sprintf("unexpected type %s", v.Type()))
	}
	return typ
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

func smt(pass *analysis.Pass) (interface{}, error) {
	// XXX we really can't use this until we have a way to differentiate literals from named consts. we're finding
	// impossible conditions that are debugging consts…

	// XXX detect and handle loops

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

func Expand(c Component, predicates map[ir.Value]Component) Component {
	switch c := c.(type) {
	case And:
		out := make(And, len(c))
		for i, k := range c {
			out[i] = Expand(k, predicates)
		}
		return out
	case Or:
		out := make(Or, len(c))
		for i, k := range c {
			out[i] = Expand(k, predicates)
		}
		return out
	case BinaryExpression:
		return BinaryExpression{
			X:  Expand(c.X, predicates),
			Op: c.Op,
			Y:  Expand(c.Y, predicates),
		}
	case Ref:
		return Expand(predicates[c.Value], predicates)
	default:
		return c
	}
}

func substitute(in Component, rename *renaming) Component {
	switch in := in.(type) {
	case And:
		out := make(And, len(in))
		for i, k := range in {
			out[i] = substitute(k, rename)
		}
		return out
	case Or:
		out := make(Or, len(in))
		for i, k := range in {
			out[i] = substitute(k, rename)
		}
		return out
	case SMTConstant:
		return in
	case SMTValue:
		if subst, _ := rename.get(in); subst != nil {
			return subst
		} else {
			return in
		}
	case BinaryExpression:
		if in.Op == token.EQL || in.Op == token.ASSIGN {
			if x, ok := in.X.(SMTValue); ok {
				x_, last := rename.get(x)
				if x_ == nil || (last && x_.Equal(in.Y)) {
					// Don't replace (= x y) with (= y y) unless definition of x came from a higher level
					x_ = x
				}
				return BinaryExpression{
					X:  x_,
					Op: in.Op,
					Y:  substitute(in.Y, rename),
				}
			} else {
				return BinaryExpression{
					X:  substitute(in.X, rename),
					Op: in.Op,
					Y:  substitute(in.Y, rename),
				}
			}
		} else {
			return BinaryExpression{
				X:  substitute(in.X, rename),
				Op: in.Op,
				Y:  substitute(in.Y, rename),
			}
		}
	default:
		panic(fmt.Sprintf("unexpected type %T", in))
	}
}

type renaming struct {
	maps []map[SMTValue]Component
}

func (r *renaming) set(v SMTValue, c Component) {
	if k, _ := r.get(v); k == nil {
		r._map()[v] = c
	}
}

func (r *renaming) get(v SMTValue) (k Component, lastLevel bool) {
	for i, m := range r.maps {
		if c, ok := m[v]; ok {
			return c, i == len(r.maps)-1
		}
	}
	return nil, false
}

func (r *renaming) _map() map[SMTValue]Component {
	return r.maps[len(r.maps)-1]
}

func (r *renaming) push() {
	r.maps = append(r.maps, map[SMTValue]Component{})
}

func (r *renaming) pop() {
	r.maps = r.maps[:len(r.maps)-1]
}

func Simplify(c Component, rename *renaming) Component {
	if rename == nil {
		rename = &renaming{}
	}
	rename.push()
	defer rename.pop()

	switch c := c.(type) {
	case And:
		o := make(And, 0, len(c))
		o = flattenAnd(c, o)

		j := 0
		for _, k := range o {
			switch k {
			case SMTConstant{Value: constant.MakeBool(true)}:
				// Meaningless term
			case SMTConstant{Value: constant.MakeBool(false)}:
				// (and ... false ...) == false
				return SMTConstant{Value: constant.MakeBool(false)}
			default:
				o[j] = Simplify(k, rename)
				j++

				if k, ok := k.(BinaryExpression); ok && (k.Op == token.EQL || k.Op == token.ASSIGN) {
					if x, ok := k.X.(SMTValue); ok {
						rename.set(x, k.Y)
					} else if y, ok := k.Y.(SMTValue); ok {
						rename.set(y, k.X)
					}
				}
			}
		}
		o = o[:j]

		for i, k := range o {
			if k == nil {
				// XXX why does this happen?
				return SMTConstant{Value: constant.MakeBool(true)}
			}
			o[i] = substitute(k, rename)
		}

		if len(o) == 0 {
			return SMTConstant{Value: constant.MakeBool(true)}
		} else if len(o) == 1 {
			return o[0]
		} else if !o.Equal(c) {
			return Simplify(o, nil)
		} else {
			return o
		}
	case Or:
		o := make(Or, 0, len(c))
		o = flattenOr(c, o)

		j := 0
		for _, k := range o {
			switch k {
			case SMTConstant{Value: constant.MakeBool(true)}:
				// (or ... true ...) == true
				return SMTConstant{Value: constant.MakeBool(true)}
			case SMTConstant{Value: constant.MakeBool(false)}:
				// Meaningless term
			default:
				o[j] = Simplify(k, rename)
				j++
			}
		}
		o = o[:j]

		if len(o) == 0 {
			return SMTConstant{Value: constant.MakeBool(false)}
		} else if len(o) == 1 {
			return o[0]
		} else if !o.Equal(c) {
			return Simplify(o, nil)
		} else {
			return o
		}
	case BinaryExpression:
		k := BinaryExpression{
			X:  Simplify(c.X, rename),
			Op: c.Op,
			Y:  Simplify(c.Y, rename),
		}

		if x, ok := k.X.(SMTConstant); ok {
			if y, ok := k.Y.(SMTConstant); ok {
				switch k.Op {
				case token.EQL, token.ASSIGN, token.LSS, token.GTR, token.LEQ, token.GEQ:
					if constant.Compare(x.Value, k.Op, y.Value) {
						return SMTConstant{Value: constant.MakeBool(true)}
					} else {
						return SMTConstant{Value: constant.MakeBool(false)}
					}
				}
				// XXX not all binary expressions are comparisons. some are math.
				// XXX once we do math, guard against division by 0
			}
		}

		switch k.Op {
		case token.EQL, token.ASSIGN, token.GEQ, token.LEQ:
			if k.X.Equal(k.Y) {
				return SMTConstant{Value: constant.MakeBool(true)}
			}
		case token.LSS, token.GTR, token.NEQ:
			if k.X.Equal(k.Y) {
				return SMTConstant{Value: constant.MakeBool(false)}
			}
		case token.ADD:
			if x, ok := c.X.(SMTConstant); ok && constant.Compare(x.Value, token.EQL, constant.MakeInt64(0)) {
				return c.Y
			} else if y, ok := c.Y.(SMTConstant); ok && constant.Compare(y.Value, token.EQL, constant.MakeInt64(0)) {
				return c.X
			}
		}

		if !k.Equal(c) {
			return Simplify(k, rename)
		} else {
			return k
		}
	default:
		return c
	}
}

// TODO: rewrite (= (+ x y) x) to (= y 0)
