// Package vrp implements value range analysis on Go programs in SSI form.
//
// Our implementation uses an iterative fixpoint algorithm on an interval lattice, with jump-set widening, and futures
// as presented in the paper "Speed And Precision in Range Analysis" by Campos et al. It is a non-relational and sparse
// analysis on SSI form.
//
// Propagating new information to old instructions
//
// Some implementations of VRP rely on an IR that is similar to eSSA, which renames variables that are being used in
// conditionals, which allows associating information with them that holds for individual branches.
//
// Consider the following piece of code:
//
// 	if x == 5 {
// 		if y == 6 {
// 			z := x + y
// 			println(z)
// 		}
// 	}
//
// With eSSA-style renaming, it will translate to
//
// 	if x == 5 {
// 		x1 := σ(x)
// 		if y == 6 {
// 			y1 := σ(y)
// 			z := x1 + y1
// 			println(z)
// 		}
// 	}
//
// We can then say that x1 ∈ [5, 5], y1 ∈ [6, 6], and z = x1 + y1 ∈ [11, 11].
//
// Now consider this slightly different example:
//
// 	z := x + y
// 	if x == 5 {
// 		if y == 6 {
// 			println(z)
// 		}
// 	}
//
// Because x and y aren't used further, and z isn't part of any conditionals, no renaming occurs and we cannot associate
// any per-branch information with the variables. Furthermore, when we evaluate z = x + y, no information is known about
// x and y, and z's bounds are [-∞, ∞].
//
// SSI, unlike eSSA, renames all variables that are used in branches, not just those that are part of conditionals. For
// the first example, while the exact translation differs, the end result is the same: we associate information with x1 and
// y1 and use these in the computation of z. However, for the second example, we end up with the following translation:
//
// 	z := x + y
// 	if x == 5 {
// 		z1 := σ(z)
// 		y1 := σ(y)
// 		if y1 == 6 {
// 			z2 := σ(z1)
// 			println(z2)
// 		}
// 	}
//
// We still have no useful σ nodes for x or y, but we do have nodes for z. This allows us to associate new information
// with z in the branches. If we associate x ∈ [5, 5] with z1 and y ∈ [6, 6] with z2, then we can reevaluate x + y
// inside the branches and end up with z1 ∈ [5, ∞] and z2 ∈ [11, 11].
//
// Note that this reevaluation is not recursive. For
//
// 	y := 2 * x
// 	z := y + 1
// 	if x == 5 {
// 		z1 := σ(z)
// 		println(z1)
// 	}
//
// z1 will not have useful bounds, because z doesn't use x. This avoids reevaluating large parts of a function multiple
// times, as well as having to deal with loops in the dataflow graph. Reevaluating more instructions would have the same
// effect as introducing more variables and is equivalent to making the analysis more dense.
package vrp

// TODO do reevaluate recursively in _some_ cases. do it when there are no cycles, and do it in a way that doesn't
// result in O(n²) - that is, don't reevaluate nodes whose inputs haven't changed.

// XXX right now our results aren't stable and change depending on the order in which we iterate over maps. why?

// OPT: constants have fixed intervals, they don't need widening or narrowing or fixpoints

// TODO: support more than one interval per value. For example, we should be able to represent the set {0, 10, 100}
// without ending up with [0, 100].

// Our handling of overflow is poor. We basically use saturated integers and when x <op> y overflows, it will be set to
// -∞ or ∞, depending if it's the lower or upper bound of an interval. This means that an interval like [1, ∞] for a
// signed integer really means that the value can be anywhere between its minimum and maximum value. For example, for a
// int8, [1, ∞] really means [-128, 127]. The reason we use [1, ∞] and not [-128, 127] or [-∞, ∞] is that our intervals
// encode growth in the lattice of intervals. In our case, the value only ever overflowed because the upper bound
// overflowed. Note that it is possible for [-∞, -100] - 100 to result in [-∞, ∞], which makes sense in the lattice, but
// doesn't really encode how the overflow happens at runtime.
//
// Nevertheless, if we used more than one interval per variable we could encode tighter bounds. For example, [5, 127] +
// 1 ought to be [6, 127] ∪ [-128, -128].

// TODO: track if constants came from literals or from named consts, to know if build tags could affect them. then
// include that information in intervals derived from constants.

import (
	"fmt"
	"go/token"
	"go/types"
	"math"
	"sort"
	"strings"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"
)

const debug = true

func Keys(m map[ir.Value]struct{}) []ir.Value {
	keys := make([]ir.Value, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func SortedKeys(m map[ir.Value]struct{}, less func(a, b ir.Value) bool) []ir.Value {
	keys := Keys(m)
	sort.Slice(keys, func(i, j int) bool {
		return less(keys[i], keys[j])
	})
	return keys
}

type Interval struct {
	Lower, Upper *Int
}

func NewInterval(l, u *Int) Interval {
	if l == nil && u != nil || l != nil && u == nil {
		panic("inconsistent interval")
	}

	return Interval{l, u}
}

func (ival Interval) Empty() bool {
	if ival.Undefined() {
		return false
	}
	if ival.Upper.Cmp(ival.Lower) == -1 {
		return true
	}
	return false
}

// XXX rename this method; it's not a traditional interval union, in which [1, 2] ∪ [4, 5] would be {1, 2, 4, 5}, not [1, 5]
func (ival Interval) Union(oval Interval) Interval {
	if ival.Empty() {
		return oval
	} else if oval.Empty() {
		return ival
	} else if ival.Undefined() {
		return oval
	} else if oval.Undefined() {
		return ival
	} else {
		return NewInterval(min(ival.Lower, oval.Lower), max(ival.Upper, oval.Upper))
	}
}

func (ival Interval) Intersect(oval Interval) Interval {
	if ival.Empty() || oval.Empty() {
		return Interval{Inf, NegInf}
	}
	if ival.Undefined() {
		return oval
	}
	if oval.Undefined() {
		return ival
	}

	return NewInterval(max(ival.Lower, oval.Lower), min(ival.Upper, oval.Upper))
}

func (ival Interval) Equal(oval Interval) bool {
	if ival.Empty() {
		return oval.Empty()
	} else if ival.Undefined() {
		return oval.Undefined()
	} else {
		return ival.Lower.Cmp(oval.Lower) == 0 && ival.Upper.Cmp(oval.Upper) == 0
	}
}

func (ival Interval) Undefined() bool {
	if ival.Lower == nil && ival.Upper != nil || ival.Lower != nil && ival.Upper == nil {
		panic("inconsistent interval")
	}
	return ival.Lower == nil
}

func (ival Interval) String() string {
	if ival.Undefined() {
		return "[⊥, ⊥]"
	} else if ival.Empty() {
		return "∅"
	} else {
		l := ival.Lower.String()
		u := ival.Upper.String()
		return fmt.Sprintf("[%s, %s]", l, u)
	}
}

// TODO: we should be able to represent both intersections using a single type
type Intersection interface {
	String() string
	Interval() Interval
}

type BasicIntersection struct {
	interval Interval
}

func (isec BasicIntersection) String() string {
	return isec.interval.String()
}

func (isec BasicIntersection) Interval() Interval {
	return isec.interval
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

func (isec SymbolicIntersection) Interval() Interval {
	// We don't have an interval for this intersection yet. If we did, the SymbolicIntersection wouldn't exist any
	// longer and would've been replaced with a basic intersection.
	return NewInterval(nil, nil)
}

func infinity() Interval {
	return NewInterval(NegInf, Inf)
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

type valueSet map[ir.Value]struct{}

type TaggedIntersection struct {
	Variable     ir.Value
	Intersection Intersection
}

type constraintGraph struct {
	// OPT: if we wrap ir.Value in a struct with some fields, then we only need one map, which reduces the number of
	// lookups and the memory usage.

	// Map σ nodes to their intersections. In SSI form, only σ nodes will have intersections. Only conditionals
	// cause intersections, and conditionals always cause the creation of σ nodes for all relevant values.
	intersections map[*ir.Sigma]Intersection
	// Intersections that describe the range of a variable to a σ using the variable in an operation
	// XXX we can merge this with the intersections field
	intersectionsFor map[*ir.Sigma][]TaggedIntersection
	// The subset of fn's instructions that make up our constraint graph.
	nodes valueSet
	// Map instructions to computed intervals
	intervals map[ir.Value]Interval
	// The graph's strongly connected components. The list of SCCs is sorted in topological order.
	sccs []valueSet
}

func min(a, b *Int) *Int {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	if a.Cmp(b) <= 0 {
		return a
	} else {
		return b
	}
}

func max(a, b *Int) *Int {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	if a.Cmp(b) >= 0 {
		return a
	} else {
		return b
	}
}

func isInteger(typ types.Type) bool {
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return (basic.Info() & types.IsInteger) != 0
}

func minInt(typ types.Type) *Int {
	basic := typ.Underlying().(*types.Basic)
	width := intWidth(basic)
	if (basic.Info() & types.IsUnsigned) == 0 {
		var min int64 = -1 << (width - 1)
		return &Int{n: min, width: width}
	} else {
		return &Int{n: 0, width: -width}
	}
}

func maxInt(typ types.Type) *Int {
	basic := typ.Underlying().(*types.Basic)
	width := intWidth(basic)
	if (basic.Info() & types.IsUnsigned) == 0 {
		var max int64 = 1<<(width-1) - 1
		return &Int{n: max, width: width}
	} else {
		var max uint64 = 1<<width - 1
		return &Int{n: int64(max), width: -width}
	}
}

func buildConstraintGraph(fn *ir.Function) *constraintGraph {
	cg := constraintGraph{
		intersections:    map[ir.Value]Intersection{},
		intersectionsFor: map[*ir.Sigma][]TaggedIntersection{},
		nodes:            valueSet{},
		intervals:        map[ir.Value]Interval{},
	}

	for _, b := range fn.Blocks {
		enrichUses := func(v ir.Value, op token.Token, val interface{}, makeIntersection func(op token.Token, val interface{}) Intersection) {
			for lv := v; ; {
				refs := *lv.Referrers() // all uses of the variable whose value we just determined
				for i, succ := range b.Succs {
					for _, ref := range refs {
						if ref, ok := ref.(ir.Value); ok {
							if σ := succ.SigmaForRecursive(ref, b); σ != nil { // find the σ for the use, or for a σ of the use, recursively
								var isec Intersection
								if i == 0 {
									isec = makeIntersection(op, val)
								} else {
									// We're in the else branch
									isec = makeIntersection(negateToken(op), val)
								}
								cg.intersectionsFor[σ] = append(cg.intersectionsFor[σ], TaggedIntersection{lv, isec})
							}
						}
					}
				}

				// We didn't find a use for this variable, but if the variable is a σ node, then there might be
				// a use for the underlying variable. For example, in
				// 	x := foo(a)
				// 	if a > 5 {
				// 		x1 := σ(x)
				// 		a1 := σ(a)
				// 		if a1 < 10 {
				// 			x2 := σ(x1)
				// 			println(x2)
				// 		}
				// 	}
				// when we learn that a1 < 10, we won't find a use of a1, but we'll find a use of 'a'.
				if llv, ok := lv.(*ir.Sigma); ok {
					lv = llv.X
				} else {
					break
				}
			}
		}

		switch ctrl := b.Control().(type) {
		case *ir.If:
			if cond, ok := ctrl.Cond.(*ir.BinOp); ok {
				lc, _ := cond.X.(*ir.Const)
				rc, _ := cond.Y.(*ir.Const)

				if lc != nil && rc != nil {
					// Comparing two constants, which isn't interesting to us
				} else if (lc != nil && rc == nil) || (lc == nil && rc != nil) {
					// Comparing a variable with a constant
					var x ir.Value
					var k *ir.Const
					var op token.Token
					if lc != nil {
						// constant on the left side
						x = cond.Y
						k = lc
						op = flipToken(cond.Op)
					} else {
						// constant on the right side
						x = cond.X
						k = rc
						op = cond.Op
					}

					makeIntersection := func(op token.Token, val_ interface{}) Intersection {
						val := val_.(*Int)
						switch op {
						case token.LSS:
							// [-∞, k-1]
							u, of := val.Dec()
							if of {
								u = Inf
							}
							return BasicIntersection{NewInterval(minInt(x.Type()), u)}
						case token.GTR:
							// [k+1, ∞]
							l, of := val.Inc()
							if of {
								l = NegInf
							}
							return BasicIntersection{NewInterval(l, maxInt(x.Type()))}
						case token.LEQ:
							// [-∞, k]
							return BasicIntersection{NewInterval(minInt(x.Type()), val)}
						case token.GEQ:
							// [k, ∞]
							return BasicIntersection{NewInterval(val, maxInt(x.Type()))}
						case token.NEQ:
							// We cannot represent this constraint
							// [-∞, ∞]
							return BasicIntersection{infinity()}
						case token.EQL:
							// [k, k]
							return BasicIntersection{NewInterval(val, val)}
						default:
							panic(fmt.Sprintf("unhandled token %s", op))
						}
					}

					// Associate learned information with σ nodes for uses of x
					enrichUses(x, op, ConstToNumeric(k), makeIntersection)

					// Associate learned information with σ node for x
					for i, succ := range b.Succs {
						σ := succ.SigmaFor(x, b)
						if σ == nil {
							continue
						}
						val := ConstToNumeric(k)
						var isec Intersection
						if i == 0 {
							isec = makeIntersection(op, val)
						} else {
							// We're in the else branch
							isec = makeIntersection(negateToken(op), val)
						}
						cg.intersections[σ] = isec
					}
				} else if lc == nil && rc == nil {
					// Comparing two variables
					if cond.X == cond.Y {
						// Comparing variable with itself, nothing to do
					} else {
						x, y := cond.X, cond.Y
						op := cond.Op

						makeIntersection := func(op token.Token, v_ interface{}) Intersection {
							v := v_.(ir.Value)
							switch op {
							case token.LSS, token.GTR, token.LEQ, token.GEQ, token.EQL:
								return SymbolicIntersection{op, v}
							case token.NEQ:
								// We cannot represent this constraint
								return BasicIntersection{infinity()}
							default:
								panic(fmt.Sprintf("unhandled token %s", op))
							}
						}

						// Associate learned information with σ nodes for uses of x and y
						enrichUses(x, op, y, makeIntersection)
						enrichUses(y, flipToken(op), x, makeIntersection)

						// Associate learned information with σ node for x
						for i, succ := range b.Succs {
							if i == 1 {
								// We're in the else branch
								op = negateToken(op)
							}

							σ1, σ2 := succ.SigmaFor(x, b), succ.SigmaFor(y, b)
							if σ1 != nil {
								cg.intersections[σ1] = makeIntersection(op, y)
							}
							if σ2 != nil {
								cg.intersections[σ2] = makeIntersection(flipToken(op), x)
							}
						}
					}
				} else {
					panic("unreachable")
				}
			}
		}

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
		}
	}

	cg.sccs = cg.buildSCCs()
	return &cg
}

func (cg *constraintGraph) fixpoint(scc valueSet, color string, fn func(ir.Value)) {
	worklist := Keys(scc)
	for len(worklist) > 0 {
		// XXX is a LIFO okay or do we need FIFO?
		op := worklist[len(worklist)-1]
		worklist = worklist[:len(worklist)-1]
		old := cg.intervals[op]

		fn(op)

		res := cg.intervals[op]
		cg.printSCCs(op, color)
		if !old.Equal(res) {
			for _, ref := range *op.Referrers() {
				if ref, ok := ref.(ir.Value); ok && isInteger(ref.Type()) {
					if _, ok := scc[ref]; ok {
						worklist = append(worklist, ref)
					}
				}
			}
		}
	}
}

func (cg *constraintGraph) widen(op ir.Value) {
	old := cg.intervals[op]
	new := cg.eval(op, nil)

	const simple = 0
	const jumpset = 1
	const infinite = 2
	const mode = simple

	switch mode {
	case simple:
		if old.Undefined() {
			cg.intervals[op] = new
		} else if new.Lower.Cmp(old.Lower) == -1 && new.Upper.Cmp(old.Upper) == 1 {
			cg.intervals[op] = infinity()
		} else if new.Lower.Cmp(old.Lower) == -1 {
			cg.intervals[op] = NewInterval(NegInf, old.Upper)
		} else if new.Upper.Cmp(old.Upper) == 1 {
			cg.intervals[op] = NewInterval(old.Lower, Inf)
		}

	case jumpset:
		panic("not implemented")

	case infinite:
		cg.intervals[op] = NewInterval(min(old.Lower, new.Lower), max(old.Upper, new.Upper))
	}
}

func (cg *constraintGraph) narrow(op ir.Value) {
	// This block is the meet narrowing operator. Narrowing is meant to replace infinites with smaller
	// bounds, but leave other bounds alone. That is, [-∞, 10] can become [0, 10], but not [0, 9] or
	// [-∞, 9]. That's why the code below selects the _wider_ bounds for non-infinities. When the
	// widening operator is implemented correctly, then the bounds shouldn't be able to grow.

	old := cg.intervals[op]

	// OPT: if the bounds aren't able to grow, then why are we doing any comparisons/assigning new intervals? Either we
	// went from an infinity to a narrower bound, or nothing should've changed. if that is true, we don't even need to
	// call eval if no bounds are infinite.
	new := cg.eval(op, nil)

	if old.Lower == NegInf && new.Lower != NegInf {
		cg.intervals[op] = NewInterval(new.Lower, old.Upper)
	} else if old.Lower.Cmp(new.Lower) == 1 {
		cg.intervals[op] = NewInterval(new.Lower, old.Upper)
	}

	if old.Upper == Inf && new.Upper != Inf {
		cg.intervals[op] = NewInterval(old.Lower, new.Upper)
	} else if old.Upper.Cmp(new.Upper) == -1 {
		cg.intervals[op] = NewInterval(old.Lower, new.Upper)
	}
}

// XXX rename this function
func XXX(fn *ir.Function) {
	cg := buildConstraintGraph(fn)
	cg.printSCCs(nil, "")

	// XXX the paper's code "propagates" values to dependent SCCs by evaluating their constraints once, so "that the
	// next SCCs after component will have entry points to kick start the range analysis algorithm". intuitively, this
	// sounds unnecessary, but I haven't looked into what "entry points" are or why we need them. "propagating" means
	// evaluating all uses of the values in the finished SCC, and if they're σ nodes, marking them as unresolved if
	// they're undefined. "entry points" are variables with ranges that aren't unknown. is this just an optimization?

	for _, scc := range cg.sccs {
		if len(scc) == 0 {
			panic("WTF")
		}

		// OPT: select favourable entry points
		cg.fixpoint(scc, "red", cg.widen)

		// Once we've finished processing the SCC we can propagate the ranges of variables to the symbolic
		// intersections that use them.
		cg.fixIntersects(scc)

		for v := range scc {
			if cg.intervals[v].Undefined() {
				cg.intervals[v] = infinity()
			}
		}

		cg.fixpoint(scc, "green", cg.narrow)
	}

	cg.printSCCs(nil, "")
}

func (cg *constraintGraph) fixIntersects(scc valueSet) {
	// XXX rename this function
	doTheThing := func(ival Interval, σival Interval, symbIsec SymbolicIntersection) Interval {
		σivall := σival.Lower
		σivalu := σival.Upper
		if σival.Undefined() {
			σivall = NegInf
			σivalu = Inf
		}
		var newval Interval
		switch symbIsec.Op {
		case token.EQL:
			newval = ival
		case token.LEQ:
			newval = NewInterval(σivall, ival.Upper)
		case token.LSS:
			// XXX the branch isn't necessary, -∞ + 1 is still -∞
			if ival.Upper != Inf {
				u, of := ival.Upper.Dec()
				if of {
					u = Inf
				}
				newval = NewInterval(σivall, u)
			} else {
				newval = NewInterval(σivall, ival.Upper)
			}
		case token.GEQ:
			newval = NewInterval(ival.Lower, σivalu)
		case token.GTR:
			// XXX the branch isn't necessary, -∞ + 1 is still -∞
			if ival.Lower != NegInf {
				l, of := ival.Lower.Inc()
				if of {
					l = NegInf
				}
				newval = NewInterval(l, σivalu)
			} else {
				newval = NewInterval(ival.Lower, σivalu)
			}
		default:
			panic(fmt.Sprintf("unhandled token %s", symbIsec.Op))
		}
		return newval
	}

	// OPT cache this compuation. also, similar code exists in buildSCCs.
	futuresUsedBy := map[ir.Value][]*ir.Sigma{}
	for σ, isec := range cg.intersections {
		if isec, ok := isec.(SymbolicIntersection); ok {
			futuresUsedBy[isec.Value] = append(futuresUsedBy[isec.Value], σ)
		}
	}
	for v := range scc {
		ival := cg.intervals[v]
		for _, σ := range futuresUsedBy[v] {
			// XXX is there any point in σival? We'll end up with σ ∩ ival, which gets evaluated at some point, so
			// doing σival ∩ ival in this step seems pointless?
			σival := cg.intervals[σ]
			symbIsec := cg.intersections[σ].(SymbolicIntersection)
			newval := doTheThing(ival, σival, symbIsec)
			cg.intersections[σ] = BasicIntersection{interval: newval}
		}
	}

	// XXX rename this variable
	futuresUsedByOther := map[ir.Value][]*TaggedIntersection{}
	for _, tsecs := range cg.intersectionsFor {
		for i := range tsecs {
			tsec := &tsecs[i]
			if isec, ok := tsec.Intersection.(SymbolicIntersection); ok {
				futuresUsedByOther[isec.Value] = append(futuresUsedByOther[isec.Value], tsec)
			}
		}
	}
	for v := range scc {
		ival := cg.intervals[v]
		for _, tsec := range futuresUsedByOther[v] {
			symbIsec := tsec.Intersection.(SymbolicIntersection)
			newval := doTheThing(ival, Interval{}, symbIsec)
			tsec.Intersection = BasicIntersection{interval: newval}
		}
	}
}

func (cg *constraintGraph) printSCCs(activeOp ir.Value, color string) {
	if !debug {
		return
	}

	// We first create subgraphs containing the nodes, then create edges between nodes. Graphviz creates a node the
	// first time it sees it, so doing 'a -> b' in a subgraph would create 'b' in that subgraph, even if it belongs in a
	// different one.
	fmt.Println("digraph{")
	n := 0
	for _, scc := range cg.sccs {
		n++
		fmt.Printf("subgraph cluster_%d {\n", n)
		for _, node := range SortedKeys(scc, func(a, b ir.Value) bool { return a.ID() < b.ID() }) {
			extra := ""
			if node == activeOp {
				extra = ", color=" + color
			}
			if σ, ok := node.(*ir.Sigma); ok {
				var ovs []string
				for _, ov := range cg.intersectionsFor[σ] {
					ovs = append(ovs, fmt.Sprintf("%s ∩ %s", ov.Variable.Name(), ov.Intersection))
				}
				isec := cg.intersections[σ]
				if isec == nil {
					isec = BasicIntersection{
						interval: infinity(),
					}
				}
				fmt.Printf("%s [label=\"%s = %s ∩ %s ← {%s} ∈ %s\"%s];\n", node.Name(), node.Name(), node, isec, strings.Join(ovs, ", "), cg.intervals[node], extra)
			} else {
				fmt.Printf("%s [label=\"%s = %s ∈ %s\"%s];\n", node.Name(), node.Name(), node, cg.intervals[node], extra)
			}
		}
		fmt.Println("}")
	}
	for _, scc := range cg.sccs {
		for _, node := range SortedKeys(scc, func(a, b ir.Value) bool { return a.ID() < b.ID() }) {
			for _, ref_ := range *node.Referrers() {
				ref, ok := ref_.(ir.Value)
				if !ok {
					continue
				}
				if _, ok := cg.nodes[ref]; !ok {
					continue
				}
				fmt.Printf("%s -> %s\n", node.Name(), ref.Name())
			}
			if node, ok := node.(*ir.Sigma); ok {
				if isec, ok := cg.intersections[node].(SymbolicIntersection); ok {
					fmt.Printf("%s -> %s [style=dashed]\n", isec.Value.Name(), node.Name())
				}
				for _, tsec := range cg.intersectionsFor[node] {
					if isec, ok := tsec.Intersection.(SymbolicIntersection); ok {
						fmt.Printf("%s -> %s [style=dashed]\n", isec.Value.Name(), node.Name())
					}
				}
			}
		}
	}
	fmt.Println("}")
}

// sccs returns the constraint graph's strongly connected components, in topological order.
func (cg *constraintGraph) buildSCCs() []valueSet {
	futuresUsedBy := map[ir.Value][]*ir.Sigma{}
	for σ, isec := range cg.intersections {
		if isec, ok := isec.(SymbolicIntersection); ok {
			futuresUsedBy[isec.Value] = append(futuresUsedBy[isec.Value], σ)
		}
	}
	for σ, isecs := range cg.intersectionsFor {
		for _, isec := range isecs {
			if isec, ok := isec.Intersection.(SymbolicIntersection); ok {
				futuresUsedBy[isec.Value] = append(futuresUsedBy[isec.Value], σ)
			}
		}
	}
	index := uint64(1)
	S := []ir.Value{}
	data := map[ir.Value]*struct {
		index   uint64
		lowlink uint64
		onstack bool
	}{}
	var sccs []valueSet

	min := func(a, b uint64) uint64 {
		if a < b {
			return a
		}
		return b
	}

	var strongconnect func(v ir.Value)
	strongconnect = func(v ir.Value) {
		vd, ok := data[v]
		if !ok {
			vd = &struct {
				index   uint64
				lowlink uint64
				onstack bool
			}{}
			data[v] = vd
		}
		vd.index = index
		vd.lowlink = index
		index++
		S = append(S, v)
		vd.onstack = true

		// XXX deduplicate code
		for _, w := range futuresUsedBy[v] {
			if _, ok := cg.nodes[w]; !ok {
				continue
			}
			wd, ok := data[w]
			if !ok {
				wd = &struct {
					index   uint64
					lowlink uint64
					onstack bool
				}{}
				data[w] = wd
			}

			if wd.index == 0 {
				strongconnect(w)
				vd.lowlink = min(vd.lowlink, wd.lowlink)
			} else if wd.onstack {
				vd.lowlink = min(vd.lowlink, wd.lowlink)
			}
		}
		for _, w_ := range *v.Referrers() {
			w, ok := w_.(ir.Value)
			if !ok {
				continue
			}
			if _, ok := cg.nodes[w]; !ok {
				continue
			}
			wd, ok := data[w]
			if !ok {
				wd = &struct {
					index   uint64
					lowlink uint64
					onstack bool
				}{}
				data[w] = wd
			}

			if wd.index == 0 {
				strongconnect(w)
				vd.lowlink = min(vd.lowlink, wd.lowlink)
			} else if wd.onstack {
				vd.lowlink = min(vd.lowlink, wd.lowlink)
			}
		}

		if vd.lowlink == vd.index {
			scc := valueSet{}
			for {
				w := S[len(S)-1]
				S = S[:len(S)-1]
				data[w].onstack = false
				scc[w] = struct{}{}
				if w == v {
					break
				}
			}
			if len(scc) > 0 {
				sccs = append(sccs, scc)
			}
		}
	}

	for v := range cg.nodes {
		if data[v] == nil || data[v].index == 0 {
			strongconnect(v)
		}
	}

	// The output of Tarjan is in reverse topological order. Reverse it to bring it into topological order.
	for i := 0; i < len(sccs)/2; i++ {
		sccs[i], sccs[len(sccs)-i-1] = sccs[len(sccs)-i-1], sccs[i]
	}

	return sccs
}

func (cg *constraintGraph) intervalFor(x ir.Value, overrides []TaggedIntersection) Interval {
	ival := cg.intervals[x]
	for _, ov := range overrides {
		if ov.Variable == x {
			ival = ival.Intersect(ov.Intersection.Interval())
		}
	}
	return ival
}

func (cg *constraintGraph) eval(v ir.Value, overrides []TaggedIntersection) Interval {
	// XXX make use of overrides in all instructions, not just BinOp

	switch v := v.(type) {
	case *ir.Const:
		n := ConstToNumeric(v)
		return NewInterval(n, n)

	case *ir.BinOp:
		xval := cg.intervalFor(v.X, overrides)
		yval := cg.intervalFor(v.Y, overrides)
		// log.Println(v, overrides)

		if xval.Undefined() || yval.Undefined() {
			return NewInterval(nil, nil)
		}

		switch v.Op {
		// XXX so much to implement
		case token.ADD:
			xl := xval.Lower
			xu := xval.Upper
			yl := yval.Lower
			yu := yval.Upper

			l := NegInf
			u := Inf
			var of bool
			if xl != NegInf && yl != NegInf {
				l, of = xl.Add(yl)
				if of {
					l = NegInf
				}
			}

			if xu != Inf && yu != Inf {
				u, of = xu.Add(yu)
				if of {
					u = Inf
				}
			}

			return NewInterval(l, u)

		case token.SUB:
			if xval.Undefined() || yval.Undefined() {
				return NewInterval(nil, nil)
			}

			xl := xval.Lower
			xu := xval.Upper
			yl := yval.Lower
			yu := yval.Upper

			var l, u *Int
			var of bool
			if xl == NegInf || yu == Inf {
				l = NegInf
			} else {
				l, of = xl.Sub(yu)
				if of {
					l = NegInf
				}
			}

			if xu == Inf || yl == NegInf {
				u = Inf
			} else {
				u, of = xu.Sub(yl)
				if of {
					u = Inf
				}
			}

			return NewInterval(l, u)

		case token.MUL:
			if xval.Undefined() || yval.Undefined() {
				return NewInterval(nil, nil)
			}

			x1 := xval.Lower
			x2 := xval.Upper
			y1 := yval.Lower
			y2 := yval.Upper

			var l, u *Int
			n1, of1 := x1.Mul(y1)
			n2, of2 := x1.Mul(y2)
			n3, of3 := x2.Mul(y1)
			n4, of4 := x2.Mul(y2)
			if of1 || of2 || of3 || of4 {
				l = NegInf
				u = Inf
			} else {
				l = min(min(n1, n2), min(n3, n4))
				u = max(max(n1, n2), max(n3, n4))
			}

			return NewInterval(l, u)

		default:
			panic(fmt.Sprintf("unhandled token %s", v.Op))
		}

	case *ir.Phi:
		ret := cg.intervals[v.Edges[0]]
		for _, other := range v.Edges[1:] {
			ret = ret.Union(cg.intervals[other])
		}
		return ret

	case *ir.Sigma:
		var ovs []TaggedIntersection
		ovs = append(ovs, overrides...)
		ovs = append(ovs, cg.intersectionsFor[v]...)
		var ival Interval
		if len(ovs) > 0 {
			ival = cg.eval(v.X, ovs)
		} else {
			ival = cg.intervals[v.X]
		}

		if ival.Undefined() {
			// If σ gets evaluated before σ.X we don't want to return the σ's intersection, which might be
			// [-∞, ∞] and saturate all instructions using the σ.
			//
			// XXX does doing this ever lose us precision?
			return NewInterval(nil, nil)
		}

		if isec, ok := cg.intersections[v]; ok {
			return ival.Intersect(isec.Interval())
		} else {
			return ival
		}

	case *ir.Parameter:
		return NewInterval(minInt(v.Type()), maxInt(v.Type()))

	case *ir.Load:
		return NewInterval(minInt(v.Type()), maxInt(v.Type()))

	case *ir.Call:
		const minInt31 = -1 << 30
		const minInt63 = -1 << 62
		const maxInt31 = 1<<30 - 1
		const maxInt63 = 1<<62 - 1

		// TODO: handle builtin len/cap

		upperMinusOne := func(cg *constraintGraph, v ir.Value) Interval {
			val := cg.intervals[v]
			if val.Undefined() {
				return NewInterval(nil, nil)
			} else if val.Empty() {
				return NewInterval(Inf, NegInf)
			} else {
				u, of := val.Upper.Dec()
				if of {
					u = Inf
				}
				width := intWidth(v.Type())
				return NewInterval(&Int{n: 0, width: width}, u)
			}
		}

		switch irutil.CallName(v.Common()) {
		case "bytes.Index", "bytes.IndexAny", "bytes.IndexByte",
			"bytes.IndexFunc", "bytes.IndexRune", "bytes.LastIndex",
			"bytes.LastIndexAny", "bytes.LastIndexByte", "bytes.LastIndexFunc",
			"strings.Index", "strings.IndexAny", "strings.IndexByte",
			"strings.IndexFunc", "strings.IndexRune", "strings.LastIndex",
			"strings.LastIndexAny", "strings.LastIndexByte", "strings.LastIndexFunc":
			// XXX don't pretend that everything uses 64 bit
			// TODO: limit to the length of the string or slice
			return NewInterval(&Int{n: -1, width: 64}, &Int{n: math.MaxInt64, width: 64})
		case "bytes.Compare", "strings.Compare":
			// XXX don't pretend that everything uses 64 bit
			// TODO: take string lengths into consideration
			return NewInterval(&Int{n: -1, width: 64}, &Int{n: 1, width: 64})
		case "bytes.Count", "strings.Count":
			// XXX don't pretend that everything uses 64 bit
			// TODO: limit to the length of the string or slice
			return NewInterval(&Int{n: -1, width: 64}, &Int{n: math.MaxInt64, width: 64})
		case "(*bytes.Buffer).Cap", "(*bytes.Buffer).Len", "(*bytes.Reader).Len", "(*bytes.Reader).Size":
			// XXX don't pretend that everything uses 64 bit
			return NewInterval(&Int{n: 0, width: 64}, &Int{n: math.MaxInt64, width: 64})

		case "math/rand.Int":
			// XXX don't pretend that everything uses 64 bit
			return NewInterval(&Int{n: 0, width: 64}, &Int{n: maxInt63, width: 64})
		case "math/rand.Int31":
			return NewInterval(&Int{n: 0, width: 32}, &Int{n: maxInt31, width: 32})
		case "math/rand.Int31n":
			// XXX handle the case where n > 31 bits
			return upperMinusOne(cg, v.Call.Args[0])
		case "math/rand.Int63":
			return NewInterval(&Int{n: 0, width: 64}, &Int{n: maxInt63, width: 64})
		case "math/rand.Int63n":
			// XXX handle the case where n > 63 bits
			return upperMinusOne(cg, v.Call.Args[0])
		case "math/rand.Intn":
			// XXX handle the case where n > 31 bits
			// XXX don't pretend that everything uses 64 bit
			return upperMinusOne(cg, v.Call.Args[0])
		case "math/rand.Uint32":
			return NewInterval(&Int{n: 0, width: -32}, &Int{n: math.MaxUint32, width: -32})
		case "math/rand.Uint64":
			m := uint64(math.MaxUint64)
			return NewInterval(&Int{n: 0, width: -64}, &Int{n: int64(m), width: -64})
		case "(*math/rand.Rand).Int":
			// XXX don't pretend that everything uses 64 bit
			return NewInterval(&Int{n: 0, width: 64}, &Int{n: maxInt63, width: 64})
		case "(*math/rand.Rand).Int31":
			return NewInterval(&Int{n: 0, width: 32}, &Int{n: maxInt31, width: 32})
		case "(*math/rand.Rand).Int31n":
			// XXX handle the case where n > 31 bits
			return upperMinusOne(cg, v.Call.Args[0])
		case "(*math/rand.Rand).Int63":
			return NewInterval(&Int{n: 0, width: 64}, &Int{n: maxInt63, width: 64})
		case "(*math/rand.Rand).Int63n":
			// XXX handle the case where n > 63 bits
			return upperMinusOne(cg, v.Call.Args[0])
		case "(*math/rand.Rand).Intn":
			// XXX don't pretend that everything uses 64 bit
			return upperMinusOne(cg, v.Call.Args[0])
		case "(*math/rand.Rand).Uint32":
			return NewInterval(&Int{n: 0, width: -32}, &Int{n: math.MaxUint32, width: -32})
		case "(*math/rand.Rand).Uint64":
			m := uint64(math.MaxUint64)
			return NewInterval(&Int{n: 0, width: -64}, &Int{n: int64(m), width: -64})
		case "(*math/rand.Zipf).Uint64":
			// TODO: we could track the creation of the Zipf instance, which determines the maximum value
			m := uint64(math.MaxUint64)
			return NewInterval(&Int{n: 0, width: -64}, &Int{n: int64(m), width: -64})
		default:
			return NewInterval(minInt(v.Type()), maxInt(v.Type()))
		}

	default:
		panic(fmt.Sprintf("unhandled type %T", v))
	}
}
