// Package sparse provides types and functions for implementing sparse
// data-flow analyses.
package sparse

import (
	"cmp"
	"fmt"
	"log"
	"math/bits"
	"slices"
	"sync"

	"honnef.co/go/tools/analysis/dfa"
	"honnef.co/go/tools/go/ir"

	"golang.org/x/exp/constraints"
)

const debugging = false

func debugf(f string, args ...any) {
	if debugging {
		log.Printf(f, args...)
	}
}

// Mapping maps a single [ir.Value] to an abstract state.
type Mapping[Elem any] struct {
	Value    ir.Value
	State    Elem
	Decision Decision
}

// Decision describes how a mapping from an [ir.Value] to an abstract state
// came to be. Decisions are provided by transfer functions when they create
// mappings.
type Decision struct {
	// The relevant values that the transfer function used to make the
	// decision.
	Inputs []ir.Value
	// A human-readable description of the decision.
	Description string
	// Whether this is the source of an abstract state. For example, in a taint
	// analysis, the call to a function that produces a tainted value would be
	// the source of the taint state, and any instructions that operate on and
	// propagate tainted values would not be sources.
	Source bool
}

func (m Mapping[Elem]) String() string {
	return fmt.Sprintf("%s = %v", m.Value.Name(), m.State)
}

// M is a helper for constructing instances of [Mapping].
func M[Elem any](v ir.Value, s Elem, d Decision) Mapping[Elem] {
	return Mapping[Elem]{Value: v, State: s, Decision: d}
}

// Ms is a helper for constructing slices of mappings.
//
// Example:
//
//	Ms(M(v1, d1, ...), M(v2, d2, ...))
func Ms[Elem comparable](ms ...Mapping[Elem]) []Mapping[Elem] {
	return ms
}

func Forward[L dfa.Semilattice[Elem], Elem any](
	fn *ir.Function,
	transfer func(*Instance[L, Elem], ir.Instruction) []Mapping[Elem],
) *Instance[L, Elem] {
	ins := &Instance[L, Elem]{
		Transfer: transfer,
		Mapping:  map[ir.Value]Mapping[Elem]{},
	}
	ins.Forward(fn)
	return ins
}

// Instance is an instance of a data-flow analysis. It is created by
// [Framework.Forward].
type Instance[L dfa.Semilattice[Elem], Elem any] struct {
	l L

	Transfer func(*Instance[L, Elem], ir.Instruction) []Mapping[Elem]
	// Mapping is the result of the analysis. Consider using Instance.Value
	// instead of accessing Mapping directly, as it correctly returns ⊥ for
	// missing values.
	Mapping map[ir.Value]Mapping[Elem]
}

// Set maps v to the abstract value d. It does not apply any checks. This
// should only be used before calling [Instance.Forward], to set initial states
// of values.
func (ins *Instance[L, Elem]) Set(v ir.Value, d Elem) {
	ins.Mapping[v] = Mapping[Elem]{Value: v, State: d}
}

// Value returns the abstract value for v. If none was set, it returns ⊥.
func (ins *Instance[L, Elem]) Value(v ir.Value) Elem {
	m, ok := ins.Mapping[v]
	if ok {
		return m.State
	} else {
		return ins.l.Ident()
	}
}

// Decision returns the decision of the mapping for v, if any.
func (ins *Instance[L, Elem]) Decision(v ir.Value) Decision {
	return ins.Mapping[v].Decision
}

var dfsDebugMu sync.Mutex

// Forward runs a forward data-flow analysis on fn.
func (ins *Instance[L, Elem]) Forward(fn *ir.Function) {
	if debugging {
		dfsDebugMu.Lock()
		defer dfsDebugMu.Unlock()
	}

	debugf("Analyzing %s\n", fn)
	if ins.Mapping == nil {
		ins.Mapping = map[ir.Value]Mapping[Elem]{}
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

		var ds []Mapping[Elem]
		if phi, ok := instr.(*ir.Phi); ok {
			d := ins.l.Ident()
			for _, edge := range phi.Edges {
				a, b := d, ins.Value(edge)
				d = ins.l.Merge(a, b)
				debugf("join(%v, %v) = %v", a, b, d)
			}
			ds = []Mapping[Elem]{
				{
					Value: phi,
					State: d,
					Decision: Decision{
						Inputs:      phi.Edges,
						Description: "this variable merges the results of multiple branches",
					},
				},
			}
		} else {
			ds = ins.Transfer(ins, instr)
		}
		if len(ds) > 0 {
			if v, ok := instr.(ir.Value); ok {
				debugf("transfer(%s = %s) = %v", v.Name(), instr, ds)
			} else {
				debugf("transfer(%s) = %v", instr, ds)
			}
		}
		for _, d := range ds {
			old := ins.Value(d.Value)
			dd := d.State
			if !ins.l.Equals(dd, old) {
				ins.Mapping[d.Value] = Mapping[Elem]{
					Value:    d.Value,
					State:    dd,
					Decision: d.Decision,
				}

				for _, ref := range *instr.Referrers() {
					worklist[ref] = struct{}{}
				}
			}
		}
		printMapping(fn, ins.Mapping)
	}
}

// Propagate is a helper for creating a [Mapping] that propagates the abstract
// state of src to dst. The desc parameter is used as the value of
// Decision.Description.
func (ins *Instance[L, Elem]) Propagate(dst, src ir.Value, desc string) Mapping[Elem] {
	return M(dst, ins.Value(src), Decision{Inputs: []ir.Value{src}, Description: desc})
}

func (ins *Instance[L, Elem]) Transform(dst ir.Value, s Elem, src ir.Value, desc string) Mapping[Elem] {
	return M(dst, s, Decision{Inputs: []ir.Value{src}, Description: desc})
}

func printMapping[Elem any](fn *ir.Function, m map[ir.Value]Elem) {
	if !debugging {
		return
	}

	debugf("Mapping for %s:\n", fn)
	var keys []ir.Value
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b ir.Value) int {
		return cmp.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
	})
	for _, k := range keys {
		v := m[k]
		debugf("\t%v\n", v)
	}
}

// BinaryTable returns a binary operator based on the provided mapping. For
// missing pairs of values, the default value will be returned.
func BinaryTable[Elem comparable](default_ Elem, m map[[2]Elem]Elem) func(Elem, Elem) Elem {
	return func(a, b Elem) Elem {
		if d, ok := m[[2]Elem{a, b}]; ok {
			return d
		} else if d, ok := m[[2]Elem{b, a}]; ok {
			return d
		} else {
			return default_
		}
	}
}

func PowerSet[Elem constraints.Integer](all Elem) []Elem {
	out := make([]Elem, all+1)
	for i := range out {
		out[i] = Elem(i)
	}
	return out
}

func MapSet[Elem constraints.Integer](set Elem, fn func(Elem) Elem) Elem {
	bits := 64 - bits.LeadingZeros64(uint64(set))
	var out Elem
	for i := range bits {
		if b := (set & (1 << i)); b != 0 {
			out |= fn(b)
		}
	}
	return out
}

func MapCartesianProduct[Elem constraints.Integer](x, y Elem, fn func(Elem, Elem) Elem) Elem {
	bitsX := 64 - bits.LeadingZeros64(uint64(x))
	bitsY := 64 - bits.LeadingZeros64(uint64(y))

	var out Elem
	for i := range bitsX {
		for j := range bitsY {
			bx := x & (1 << i)
			by := y & (1 << j)

			if bx != 0 && by != 0 {
				out |= fn(bx, by)
			}
		}
	}

	return out
}
