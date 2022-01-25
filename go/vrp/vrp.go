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

// XXX right now our results aren't stable and change depending on the order in which we iterate over maps. why?

// OPT: constants have fixed intervals, they don't need widening or narrowing or fixpoints

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"log"
	"math/big"
	"sort"

	"honnef.co/go/tools/go/ir"
)

var Inf Numeric = Infinity{}
var NegInf Numeric = Infinity{negative: true}
var Empty = NewInterval(Inf, NegInf)
var One = Number{big.NewInt(1)}
var MinusOne = Number{big.NewInt(-1)}

func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func SortedKeys[K comparable, V any](m map[K]V, less func(a, b K) bool) []K {
	keys := Keys(m)
	sort.Slice(keys, func(i, j int) bool {
		return less(keys[i], keys[j])
	})
	return keys
}

type Numeric interface {
	isNumeric()
	Cmp(other Numeric) int
	String() string
	Negative() bool
	Add(Numeric) Numeric
	Sub(Numeric) Numeric
}

type Infinity struct {
	negative bool
}

type Number struct {
	Number *big.Int
}

func (v Infinity) Negative() bool { return v.negative }
func (v Number) Negative() bool   { return v.Number.Sign() == -1 }

func (v Infinity) Cmp(other Numeric) int { return Cmp(v, other) }
func (v Number) Cmp(other Numeric) int   { return Cmp(v, other) }

func (v Infinity) Add(other Numeric) Numeric { return Add(v, other) }
func (v Number) Add(other Numeric) Numeric   { return Add(v, other) }

func (v Infinity) Sub(other Numeric) Numeric { return Sub(v, other) }
func (v Number) Sub(other Numeric) Numeric   { return Sub(v, other) }

func (v Infinity) String() string {
	if v.negative {
		return "-∞"
	} else {
		return "∞"
	}
}

func (v Number) String() string {
	return v.Number.String()
}

func (Infinity) isNumeric() {}
func (Number) isNumeric()   {}

func Add(x, y Numeric) Numeric {
	if x, ok := x.(Infinity); ok {
		if x.negative {
			panic("-∞ + y is not defined")
		}
		return x
	}

	if y, ok := y.(Infinity); ok {
		// x + ∞ = ∞
		// x - ∞ = -∞
		return y
	}

	nx := x.(Number)
	ny := y.(Number)

	out := big.NewInt(0)
	out = out.Add(nx.Number, ny.Number)
	return Number{out}
}

func Sub(x, y Numeric) Numeric {
	if x == NegInf {
		panic("-inf - y is not defined")
	}

	// x - -inf = x + inf = inf
	if y == NegInf {
		return Inf
	}

	// x - inf = -inf
	if y == Inf {
		return NegInf
	}

	if x == Inf {
		return Inf
	}

	nx := x.(Number)
	ny := y.(Number)

	out := big.NewInt(0)
	out = out.Sub(nx.Number, ny.Number)
	return Number{out}
}

func Cmp(x, y Numeric) int {
	if x == y {
		return 0
	}
	switch x := x.(type) {
	case Infinity:
		switch y := y.(type) {
		case Infinity:
			if x.negative == y.negative {
				return 0
			} else if x.negative {
				return -1
			} else {
				return 1
			}
		case Number:
			if x.negative {
				return -1
			} else {
				return 1
			}
		default:
			panic(fmt.Sprintf("unhandled type %T", x))
		}
	case Number:
		switch y := y.(type) {
		case Infinity:
			if y.negative {
				return 1
			} else {
				return -1
			}
		case Number:
			return x.Number.Cmp(y.Number)
		default:
			panic(fmt.Sprintf("unhandled type %T", x))
		}
	default:
		panic(fmt.Sprintf("unhandled type %T", x))
	}
}

type Interval struct {
	Lower, Upper Numeric
}

func NewInterval(l, u Numeric) Interval {
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

func (ival Interval) Union(oval Interval) (RET Interval) {
	defer func() {
		log.Printf("%s ∪ %s = %s", ival, oval, RET)
	}()
	if ival.Empty() {
		return oval
	} else if oval.Empty() {
		return ival
	} else if ival.Undefined() {
		return oval
	} else if oval.Undefined() {
		return ival
	} else {
		var l, u Numeric
		if ival.Lower.Cmp(oval.Lower) == -1 {
			l = ival.Lower
		} else {
			l = oval.Lower
		}

		if ival.Upper.Cmp(oval.Upper) == 1 {
			u = ival.Upper
		} else {
			u = oval.Upper
		}

		return NewInterval(l, u)
	}
}

func (ival Interval) Intersect(oval Interval) (RET Interval) {
	defer func() {
		log.Printf("%s ∩ %s = %s", ival, oval, RET)
	}()
	if ival.Empty() || oval.Empty() {
		return Empty
	}
	if ival.Undefined() {
		return oval
	}
	if oval.Undefined() {
		return ival
	}

	var l, u Numeric
	if ival.Lower.Cmp(oval.Lower) == 1 {
		l = ival.Lower
	} else {
		l = oval.Lower
	}

	if ival.Upper.Cmp(oval.Upper) == -1 {
		u = ival.Upper
	} else {
		u = oval.Upper
	}

	return NewInterval(l, u)
}

func (ival Interval) Equal(oval Interval) bool {
	return (ival.Lower == nil && oval.Lower == nil) || (ival.Lower != nil && oval.Lower != nil) &&
		(ival.Upper == nil && oval.Upper == nil) || (ival.Upper != nil && oval.Upper != nil) &&
		(ival.Lower.Cmp(oval.Lower) == 0) &&
		(ival.Upper.Cmp(oval.Upper) == 0)
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
	// XXX should unsigned integers be [-inf, inf] or [0, inf]?
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

var one = big.NewInt(1)

type valueSet map[ir.Value]struct{}

type constraintGraph struct {
	// OPT: if we wrap ir.Value in a struct with some fields, then we only need one map, which reduces the number of
	// lookups and the memory usage.

	// Map sigma nodes to their intersections. In SSI form, only sigma nodes will have intersections. Only conditionals
	// cause intersections, and conditionals always cause the creation of sigma nodes for all relevant values.
	intersections map[*ir.Sigma]Intersection
	// The subset of fn's instructions that make up our constraint graph.
	nodes valueSet
	// Map instructions to computed intervals
	intervals map[ir.Value]Interval
	// The graph's strongly connected components. Each component is represented as a slice of values, sorted by ID. The
	// list of SCCs is sorted in topological order.
	sccs [][]ir.Value
}

func min(a, b Numeric) Numeric {
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

func max(a, b Numeric) Numeric {
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

func BuildConstraintGraph(fn *ir.Function) *constraintGraph {
	cg := constraintGraph{
		intersections: map[*ir.Sigma]Intersection{},
		nodes:         valueSet{},
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
				cg.intersections[v] = BasicIntersection{interval: infinity()}
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
								val := constantToBigInt(k.Value)
								switch op {
								case token.LSS:
									// [-∞, k-1]
									cg.intersections[v] = BasicIntersection{NewInterval(NegInf, Number{val.Sub(val, one)})}
								case token.GTR:
									// [k+1, ∞]
									cg.intersections[v] = BasicIntersection{NewInterval(Number{val.Add(val, one)}, Inf)}
								case token.LEQ:
									// [-∞, k]
									cg.intersections[v] = BasicIntersection{NewInterval(NegInf, Number{val})}
								case token.GEQ:
									// [k, ∞]
									cg.intersections[v] = BasicIntersection{NewInterval(Number{val}, Inf)}
								case token.NEQ:
									// We cannot represent this constraint
									// [-∞, ∞]
									cg.intersections[v] = BasicIntersection{infinity()}
								case token.EQL:
									// [k, k]
									cg.intersections[v] = BasicIntersection{NewInterval(Number{val}, Number{val})}
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

	cg.sccs = cg.buildSCCs()
	return &cg
}

func XXX(fn *ir.Function) {
	cg := BuildConstraintGraph(fn)
	if true {
		// cg.printConstraints()
		cg.printSCCs(nil, "")
	}

	// XXX the paper's code "propagates" values to dependent SCCs by evaluating their constraints once, so "that the
	// next SCCs after component will have entry points to kick start the range analysis algorithm". intuitively, this
	// sounds unnecessary, but I haven't looked into what "entry points" are or why we need them. "propagating" means
	// evaluating all uses of the values in the finished SCC, and if they're sigma nodes, marking them as unresolved if
	// they're undefined. "entry points" are variables with ranges that aren't unknown. is this just an optimization?

	for _, scc := range cg.sccs {
		if len(scc) == 0 {
			panic("WTF")
		}

		// OPT: use a worklist approach
		changed := true
		for changed {
			changed = false
			for _, op := range scc {
				old := cg.intervals[op]
				{
					// this block is the meet widening operator
					// XXX implement jump-set widening
					new := cg.eval(op)

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
				res := cg.intervals[op]
				// log.Printf("W: %s = %s: %s -> %s", op.Name(), op, old, res)
				cg.printSCCs(op, "red")
				if !old.Equal(res) {
					changed = true
				}
			}
		}

		// Once we've finished processing the SCC we can propagate the ranges of variables to the symbolic
		// intersections that use them.
		cg.fixIntersects(scc)

		for _, v := range scc {
			if cg.intervals[v].Undefined() {
				cg.intervals[v] = infinity()
			}
		}

		log.Println("Finished widening SCC")
		for _, v := range scc {
			log.Printf("%s = %s ∈ %s", v.Name(), v, cg.intervals[v])
		}

		changed = true
		for changed {
			changed = false
			for _, op := range scc {
				old := cg.intervals[op]
				{
					// This block is the meet narrowing operator. Narrowing is meant to replace infinites with smaller
					// bounds, but leave other bounds alone. That is, [-∞, 10] can become [0, 10], but not [0, 9] or
					// [-∞, 9]. That's why the code below selects the _wider_ bounds for non-infinities. When the
					// widening operator is implemented correctly, then the bounds shouldn't be able to grow.

					// OPT: if the bounds aren't able to grow, then why are we doing any comparisons/assigning new
					// intervals? Either we went from an infinity to a narrower bound, or nothing should've changed.
					new := cg.eval(op)

					if old.Lower == NegInf && new.Lower != NegInf {
						cg.intervals[op] = NewInterval(new.Lower, old.Upper)
					} else {
						if old.Lower.Cmp(new.Lower) == 1 {
							cg.intervals[op] = NewInterval(new.Lower, old.Upper)
						}
					}

					if old.Upper == Inf && new.Upper != Inf {
						cg.intervals[op] = NewInterval(old.Lower, new.Upper)
					} else {
						if old.Upper.Cmp(new.Upper) == -1 {
							cg.intervals[op] = NewInterval(old.Lower, new.Upper)
						}
					}
				}

				res := cg.intervals[op]
				// log.Printf("N: %s = %s: %s -> %s", op.Name(), op, old, res)
				cg.printSCCs(op, "green")
				if !old.Equal(res) {
					changed = true
				}

				// TODO: we can implement iterative narrowing that shrinks the intervals even for non-infinites. it
				// might not converge, but we can stop any time. I'm not sure if it actually works for our lattice and
				// evaluation functions, though.
			}
		}

		log.Println("Finished narrowing SCC")
		for _, v := range scc {
			log.Printf("%s = %s ∈ %s", v.Name(), v, cg.intervals[v])
		}

		log.Println("---------------------------------------------------------")
	}

	cg.printSCCs(nil, "")

	if false {
		keys := SortedKeys(cg.intervals, func(a, b ir.Value) bool { return a.ID() < b.ID() })
		for _, v := range keys {
			ival := cg.intervals[v]
			fmt.Printf("%s$=$%s$∈$%s\n", v.Name(), v, ival)
		}
	}
}

func (cg *constraintGraph) fixIntersects(scc []ir.Value) {
	// OPT cache this compuation
	futuresUsedBy := map[ir.Value][]*ir.Sigma{}
	for sigma, isec := range cg.intersections {
		if isec, ok := isec.(SymbolicIntersection); ok {
			futuresUsedBy[isec.Value] = append(futuresUsedBy[isec.Value], sigma)
		}
	}
	for _, v := range scc {
		ival := cg.intervals[v]
		for _, sigma := range futuresUsedBy[v] {
			sval := cg.intervals[sigma]
			symb := cg.intersections[sigma].(SymbolicIntersection)
			svall := sval.Lower
			svalu := sval.Upper
			if sval.Undefined() {
				svall = NegInf
				svalu = Inf
			}
			var newval Interval
			switch symb.Op {
			case token.EQL:
				newval = ival
			case token.LEQ:
				newval = NewInterval(svall, ival.Upper)
			case token.LSS:
				// XXX the branch isn't necessary, -∞ + 1 is still -∞
				if ival.Upper != Inf {
					newval = NewInterval(svall, ival.Upper.Add(MinusOne))
				} else {
					newval = NewInterval(svall, ival.Upper)
				}
			case token.GEQ:
				newval = NewInterval(ival.Lower, svalu)
			case token.GTR:
				// XXX the branch isn't necessary, -∞ + 1 is still -∞
				if ival.Lower != NegInf {
					newval = NewInterval(ival.Lower.Add(One), svalu)
				} else {
					newval = NewInterval(ival.Lower, svalu)
				}
			default:
				panic(fmt.Sprintf("unhandled token %s", symb.Op))
			}
			log.Printf("intersection of %s changed from %s to %s", sigma, cg.intersections[sigma], BasicIntersection{interval: newval})
			cg.intersections[sigma] = BasicIntersection{interval: newval}
		}
	}
}

func (cg *constraintGraph) printSCCs(activeOp ir.Value, color string) {
	// We first create subgraphs containing the nodes, then create edges between nodes. Graphviz creates a node the
	// first time it sees it, so doing 'a -> b' in a subgraph would create 'b' in that subgraph, even if it belongs in a
	// different one.
	fmt.Println("digraph{")
	n := 0
	for _, scc := range cg.sccs {
		n++
		fmt.Printf("subgraph cluster_%d {\n", n)
		for _, node := range scc {
			extra := ""
			if node == activeOp {
				extra = ", color=" + color
			}
			if sigma, ok := node.(*ir.Sigma); ok {
				fmt.Printf("%s [label=\"%s = %s ∩ %s ∈ %s\"%s];\n", node.Name(), node.Name(), node, cg.intersections[sigma], cg.intervals[node], extra)
			} else {
				fmt.Printf("%s [label=\"%s = %s ∈ %s\"%s];\n", node.Name(), node.Name(), node, cg.intervals[node], extra)
			}
		}
		fmt.Println("}")
	}
	for _, scc := range cg.sccs {
		for _, node := range scc {
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
			}
		}
	}
	fmt.Println("}")
}

// sccs returns the constraint graph's strongly connected components, in topological order.
func (cg *constraintGraph) buildSCCs() [][]ir.Value {
	futuresUsedBy := map[ir.Value][]*ir.Sigma{}
	for sigma, isec := range cg.intersections {
		if isec, ok := isec.(SymbolicIntersection); ok {
			futuresUsedBy[isec.Value] = append(futuresUsedBy[isec.Value], sigma)
		}
	}
	index := uint64(1)
	S := []ir.Value{}
	data := map[ir.Value]*struct {
		index   uint64
		lowlink uint64
		onstack bool
	}{}
	var sccs [][]ir.Value

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
			var scc []ir.Value
			for {
				w := S[len(S)-1]
				S = S[:len(S)-1]
				data[w].onstack = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			if len(scc) > 0 {
				sort.Slice(scc, func(i, j int) bool { return scc[i].ID() < scc[j].ID() })
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

func (cg *constraintGraph) printConstraints() {
	for v := range cg.nodes {
		switch v := v.(type) {
		case *ir.Sigma:
			fmt.Printf("%s = %s ∩ %s\n", v.Name(), v.X.Name(), cg.intersections[v])
		case *ir.Const:
			fmt.Printf("%s = %s\n", v.Name(), v.Value)
		default:
			fmt.Printf("%s = %s\n", v.Name(), v)
		}
	}
}

func (cg *constraintGraph) eval(v ir.Value) Interval {
	switch v := v.(type) {
	case *ir.Const:
		return NewInterval(constantToNumber(v.Value), constantToNumber(v.Value))
	case *ir.BinOp:
		xval := cg.intervals[v.X]
		yval := cg.intervals[v.Y]

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
			if xl != NegInf && yl != NegInf {
				l = xl.Add(yl)

				if xl.Negative() == yl.Negative() && xl.Negative() != l.Negative() {
					l = NegInf
				}
			}

			if xu != Inf && yu != Inf {
				u = xu.Add(yu)

				if xu.Negative() == yu.Negative() && xu.Negative() != u.Negative() {
					u = Inf
				}
			}

			return NewInterval(l, u)

		case token.SUB:
			xval := cg.intervals[v.X]
			yval := cg.intervals[v.Y]

			if xval.Undefined() || yval.Undefined() {
				return NewInterval(nil, nil)
			}

			xl := xval.Lower
			xu := xval.Upper
			yl := yval.Lower
			yu := yval.Upper

			var l, u Numeric

			if xl == NegInf || yu == Inf {
				l = NegInf
			} else {
				l = xl.Sub(yu)

				if xl.Negative() != yu.Negative() && yu.Negative() == l.Negative() {
					l = NegInf
				}
			}

			if xu == Inf || yl == NegInf {
				u = Inf
			} else {
				u = xu.Sub(yl)

				if xu.Negative() != yl.Negative() && yl.Negative() == u.Negative() {
					u = Inf
				}
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
		if cg.intervals[v.X].Undefined() {
			// If sigma gets evaluated before sigma.X we don't want to return the sigma's intersection, which might be
			// [-∞, ∞] and saturate all instructions using the sigma.
			//
			// XXX can we do this without losing precision?
			return NewInterval(nil, nil)
		}

		return cg.intervals[v.X].Intersect(cg.intersections[v].Interval())
	case *ir.Parameter:
		return infinity()
	default:
		panic(fmt.Sprintf("unhandled type %T", v))
	}
}

func constantToNumber(v constant.Value) Number {
	return Number{constantToBigInt(v)}
}

func constantToBigInt(v constant.Value) *big.Int {
	val := big.NewInt(0)
	switch v := constant.Val(v).(type) {
	case int64:
		val.SetInt64(v)
	case *big.Int:
		val.Set(v)
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
	return val
}
