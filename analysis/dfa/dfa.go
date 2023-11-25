// Package dfa provides types and functions for implementing data-flow analyses.
package dfa

import (
	"cmp"
	"fmt"
	"log"
	"math/bits"
	"slices"
	"strings"
	"sync"

	"golang.org/x/exp/constraints"
	"honnef.co/go/tools/go/ir"
)

const debugging = false

func debugf(f string, args ...any) {
	if debugging {
		log.Printf(f, args...)
	}
}

// Join defines the [∨] operation for a [join-semilattice]. It must implement a commutative and associative binary operation
// that returns the least upper bound of two states from S.
//
// Code that calls Join functions is expected to handle the [⊥ and ⊤ elements], as well as implement idempotency. That is,
// the following properties will be enforced:
//
//   - x ∨ ⊥ = x
//   - x ∨ ⊤ = ⊤
//   - x ∨ x = x
//
// Simple table-based join functions can be created using [JoinTable].
//
// [∨]: https://en.wikipedia.org/wiki/Join_and_meet
// [join-semilattice]: https://en.wikipedia.org/wiki/Semilattice
// [⊥ and ⊤ elements]: https://en.wikipedia.org/wiki/Greatest_element_and_least_element#Top_and_bottom
type Join[S comparable] func(S, S) S

// Mapping maps a single [ir.Value] to an abstract state.
type Mapping[S comparable] struct {
	Value    ir.Value
	State    S
	Decision Decision
}

// Decision describes how a mapping from an [ir.Value] to an abstract state came to be.
// Decisions are provided by transfer functions when they create mappings.
type Decision struct {
	// The relevant values that the transfer function used to make the decision.
	Inputs []ir.Value
	// A human-readable description of the decision.
	Description string
	// Whether this is the source of an abstract state. For example, in a taint analysis, the call to a function that
	// produces a tainted value would be the source of the taint state, and any instructions that operate on
	// and propagate tainted values would not be sources.
	Source bool
}

func (m Mapping[S]) String() string {
	return fmt.Sprintf("%s = %v", m.Value.Name(), m.State)
}

// M is a helper for constructing instances of [Mapping].
func M[S comparable](v ir.Value, s S, d Decision) Mapping[S] {
	return Mapping[S]{Value: v, State: s, Decision: d}
}

// Ms is a helper for constructing slices of mappings.
//
// Example:
//
//	Ms(M(v1, d1, ...), M(v2, d2, ...))
func Ms[S comparable](ms ...Mapping[S]) []Mapping[S] {
	return ms
}

// Framework describes a monotone data-flow framework ⟨S, ∨, Transfer⟩ using a bounded join-semilattice ⟨S, ∨⟩ and a
// monotonic transfer function.
//
// Transfer implements the transfer function. Given an instruction, it should return zero or more mappings from IR
// values to abstract values, i.e. values from the semilattice. Transfer must be monotonic. ϕ instructions are handled
// automatically and do not cause Transfer to be called.
//
// The set S is defined implicitly by the values returned by Join and Transfer and needn't be finite. In addition, it
// contains the elements ⊥ and ⊤ (Bottom and Top) with Join(x, ⊥) = x and Join(x, ⊤) = ⊤. The provided Join function is
// wrapped to handle these elements automatically. All IR values start in the ⊥ state.
//
// Abstract states are associated with IR values. As such, the analysis is sparse and favours the partitioned variable
// lattice (PVL) property.
type Framework[S comparable] struct {
	Join     Join[S]
	Transfer func(*Instance[S], ir.Instruction) []Mapping[S]
	Bottom   S
	Top      S
}

// Start returns a new instance of the framework. See also [Framework.Forward].
func (fw *Framework[S]) Start() *Instance[S] {
	if fw.Bottom == fw.Top {
		panic("framework's ⊥ and ⊤ are identical; did you forget to specify them?")
	}

	return &Instance[S]{
		Framework: fw,
		Mapping:   map[ir.Value]Mapping[S]{},
	}
}

// Forward runs an intraprocedural forward data flow analysis, using an iterative fixed-point algorithm, given the
// functions specified in the framework. It combines [Framework.Start] and [Instance.Forward].
func (fw *Framework[S]) Forward(fn *ir.Function) *Instance[S] {
	ins := fw.Start()
	ins.Forward(fn)
	return ins
}

// Dot returns a directed graph in [Graphviz] format that represents the finite join-semilattice ⟨S, ≤⟩.
// Vertices represent elements in S and edges represent the ≤ relation between elements.
// We map from ⟨S, ∨⟩ to ⟨S, ≤⟩ by computing x ∨ y for all elements in [S]², where x ≤ y iff x ∨ y == y.
//
// The resulting graph can be filtered through [tred] to compute the transitive reduction of the graph, the
// visualisation of which corresponds to the Hasse diagram of the semilattice.
//
// The set of states should not include the ⊥ and ⊤ elements.
//
// [Graphviz]: https://graphviz.org/
// [tred]: https://graphviz.org/docs/cli/tred/
func Dot[S comparable](fn Join[S], states []S, bottom, top S) string {
	var sb strings.Builder
	sb.WriteString("digraph{\n")
	sb.WriteString("rankdir=\"BT\"\n")

	for i, v := range states {
		if vs, ok := any(v).(fmt.Stringer); ok {
			fmt.Fprintf(&sb, "n%d [label=%q]\n", i, vs)
		} else {
			fmt.Fprintf(&sb, "n%d [label=%q]\n", i, fmt.Sprintf("%v", v))
		}
	}

	for dx, x := range states {
		for dy, y := range states {
			if dx == dy {
				continue
			}

			if join(fn, x, y, bottom, top) == y {
				fmt.Fprintf(&sb, "n%d -> n%d\n", dx, dy)
			}
		}
	}

	sb.WriteString("}")
	return sb.String()
}

// Instance is an instance of a data-flow analysis. It is created by [Framework.Forward].
type Instance[S comparable] struct {
	Framework *Framework[S]
	// Mapping is the result of the analysis. Consider using Instance.Value instead of accessing Mapping
	// directly, as it correctly returns ⊥ for missing values.
	Mapping map[ir.Value]Mapping[S]
}

// Set maps v to the abstract value d. It does not apply any checks. This should only be used before calling [Instance.Forward], to set
// initial states of values.
func (ins *Instance[S]) Set(v ir.Value, d S) {
	ins.Mapping[v] = Mapping[S]{Value: v, State: d}
}

// Value returns the abstract value for v. If none was set, it returns ⊥.
func (ins *Instance[S]) Value(v ir.Value) S {
	m, ok := ins.Mapping[v]
	if ok {
		return m.State
	} else {
		return ins.Framework.Bottom
	}
}

// Decision returns the decision of the mapping for v, if any.
func (ins *Instance[S]) Decision(v ir.Value) Decision {
	return ins.Mapping[v].Decision
}

var dfsDebugMu sync.Mutex

func join[S comparable](fn Join[S], a, b, bottom, top S) S {
	switch {
	case a == top || b == top:
		return top
	case a == bottom:
		return b
	case b == bottom:
		return a
	case a == b:
		return a
	default:
		return fn(a, b)
	}
}

// Forward runs a forward data-flow analysis on fn.
func (ins *Instance[S]) Forward(fn *ir.Function) {
	if debugging {
		dfsDebugMu.Lock()
		defer dfsDebugMu.Unlock()
	}

	debugf("Analyzing %s\n", fn)
	if ins.Mapping == nil {
		ins.Mapping = map[ir.Value]Mapping[S]{}
	}

	worklist := map[ir.Instruction]struct{}{}
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			worklist[instr] = struct{}{}
		}
	}
	for len(worklist) > 0 {
		var instr ir.Instruction
		for instr = range worklist {
			break
		}
		delete(worklist, instr)

		var ds []Mapping[S]
		if phi, ok := instr.(*ir.Phi); ok {
			d := ins.Framework.Bottom
			for _, edge := range phi.Edges {
				a, b := d, ins.Value(edge)
				d = join(ins.Framework.Join, a, b, ins.Framework.Bottom, ins.Framework.Top)
				debugf("join(%v, %v) = %v", a, b, d)
			}
			ds = []Mapping[S]{{Value: phi, State: d, Decision: Decision{Inputs: phi.Edges, Description: "this variable merges the results of multiple branches"}}}
		} else {
			ds = ins.Framework.Transfer(ins, instr)
		}
		if len(ds) > 0 {
			if v, ok := instr.(ir.Value); ok {
				debugf("transfer(%s = %s) = %v", v.Name(), instr, ds)
			} else {
				debugf("transfer(%s) = %v", instr, ds)
			}
		}
		for i, d := range ds {
			old := ins.Value(d.Value)
			dd := d.State
			if dd != old {
				if j := join(ins.Framework.Join, old, dd, ins.Framework.Bottom, ins.Framework.Top); j != dd {
					panic(fmt.Sprintf("transfer function isn't monotonic; Transfer(%v)[%d] = %v; join(%v, %v) = %v", instr, i, dd, old, dd, j))
				}
				ins.Mapping[d.Value] = Mapping[S]{Value: d.Value, State: dd, Decision: d.Decision}

				for _, ref := range *instr.Referrers() {
					worklist[ref] = struct{}{}
				}
			}
		}
		printMapping(fn, ins.Mapping)
	}
}

// Propagate is a helper for creating a [Mapping] that propagates the abstract state of src to dst.
// The desc parameter is used as the value of Decision.Description.
func (ins *Instance[S]) Propagate(dst, src ir.Value, desc string) Mapping[S] {
	return M(dst, ins.Value(src), Decision{Inputs: []ir.Value{src}, Description: desc})
}

func (ins *Instance[S]) Transform(dst ir.Value, s S, src ir.Value, desc string) Mapping[S] {
	return M(dst, s, Decision{Inputs: []ir.Value{src}, Description: desc})
}

func printMapping[S any](fn *ir.Function, m map[ir.Value]S) {
	if !debugging {
		return
	}

	debugf("Mapping for %s:\n", fn)
	var keys []ir.Value
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b ir.Value) int {
		return cmp.Compare(a.ID(), b.ID())
	})
	for _, k := range keys {
		v := m[k]
		debugf("\t%v\n", v)
	}
}

// BinaryTable returns a binary operator based on the provided mapping.
// For missing pairs of values, the default value will be returned.
func BinaryTable[S comparable](default_ S, m map[[2]S]S) func(S, S) S {
	return func(a, b S) S {
		if d, ok := m[[2]S{a, b}]; ok {
			return d
		} else if d, ok := m[[2]S{b, a}]; ok {
			return d
		} else {
			return default_
		}
	}
}

// JoinTable returns a [Join] function based on the provided mapping.
// For missing pairs of values, the default value will be returned.
func JoinTable[S comparable](top S, m map[[2]S]S) Join[S] {
	return func(a, b S) S {
		if d, ok := m[[2]S{a, b}]; ok {
			return d
		} else if d, ok := m[[2]S{b, a}]; ok {
			return d
		} else {
			return top
		}
	}
}

func PowerSet[S constraints.Integer](all S) []S {
	out := make([]S, all+1)
	for i := range out {
		out[i] = S(i)
	}
	return out
}

func MapSet[S constraints.Integer](set S, fn func(S) S) S {
	bits := 64 - bits.LeadingZeros64(uint64(set))
	var out S
	for i := 0; i < bits; i++ {
		if b := (set & (1 << i)); b != 0 {
			out |= fn(b)
		}
	}
	return out
}

func MapCartesianProduct[S constraints.Integer](x, y S, fn func(S, S) S) S {
	bitsX := 64 - bits.LeadingZeros64(uint64(x))
	bitsY := 64 - bits.LeadingZeros64(uint64(y))

	var out S
	for i := 0; i < bitsX; i++ {
		for j := 0; j < bitsY; j++ {
			bx := x & (1 << i)
			by := y & (1 << j)

			if bx != 0 && by != 0 {
				out |= fn(bx, by)
			}
		}
	}

	return out
}
