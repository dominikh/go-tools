// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ir

// This file defines the lifting pass which tries to "lift" Alloc
// cells (new/local variables) into SSA registers, replacing loads
// with the dominating stored value, eliminating loads and stores, and
// inserting φ- and σ-nodes as needed.

// Cited papers and resources:
//
// Ron Cytron et al. 1991. Efficiently computing SSA form...
// http://doi.acm.org/10.1145/115372.115320
//
// Cooper, Harvey, Kennedy.  2001.  A Simple, Fast Dominance Algorithm.
// Software Practice and Experience 2001, 4:1-10.
// http://www.hipersoft.rice.edu/grads/publications/dom14.pdf
//
// Daniel Berlin, llvmdev mailing list, 2012.
// http://lists.cs.uiuc.edu/pipermail/llvmdev/2012-January/046638.html
// (Be sure to expand the whole thread.)
//
// C. Scott Ananian. 1997. The static single information form.
//
// Jeremy Singer. 2006. Static program analysis based on virtual register renaming.

// TODO(adonovan): opt: there are many optimizations worth evaluating, and
// the conventional wisdom for SSA construction is that a simple
// algorithm well engineered often beats those of better asymptotic
// complexity on all but the most egregious inputs.
//
// Danny Berlin suggests that the Cooper et al. algorithm for
// computing the dominance frontier is superior to Cytron et al.
// Furthermore he recommends that rather than computing the DF for the
// whole function then renaming all alloc cells, it may be cheaper to
// compute the DF for each alloc cell separately and throw it away.
//
// Consider exploiting liveness information to avoid creating dead
// φ-nodes which we then immediately remove.
//
// Also see many other "TODO: opt" suggestions in the code.

import (
	"fmt"
	"go/types"
	"os"
)

// If true, show diagnostic information at each step of lifting.
// Very verbose.
const debugLifting = false

// domFrontier maps each block to the set of blocks in its dominance
// frontier.  The outer slice is conceptually a map keyed by
// Block.Index.  The inner slice is conceptually a set, possibly
// containing duplicates.
//
// TODO(adonovan): opt: measure impact of dups; consider a packed bit
// representation, e.g. big.Int, and bitwise parallel operations for
// the union step in the Children loop.
//
// domFrontier's methods mutate the slice's elements but not its
// length, so their receivers needn't be pointers.
//
type domFrontier [][]*BasicBlock

func (df domFrontier) add(u, v *BasicBlock) {
	df[u.Index] = append(df[u.Index], v)
}

// build builds the dominance frontier df for the dominator tree of
// fn, using the algorithm found in A Simple, Fast Dominance
// Algorithm, Figure 5.
//
// TODO(adonovan): opt: consider Berlin approach, computing pruned SSA
// by pruning the entire IDF computation, rather than merely pruning
// the DF -> IDF step.
func (df domFrontier) build(fn *Function) {
	for _, b := range fn.Blocks {
		if len(b.Preds) >= 2 {
			for _, p := range b.Preds {
				runner := p
				for runner != b.dom.idom {
					df.add(runner, b)
					runner = runner.dom.idom
				}
			}
		}
	}
}

func buildDomFrontier(fn *Function) domFrontier {
	df := make(domFrontier, len(fn.Blocks))
	df.build(fn)
	return df
}

type postDomFrontier [][]*BasicBlock

func (rdf postDomFrontier) add(u, v *BasicBlock) {
	rdf[u.Index] = append(rdf[u.Index], v)
}

func (rdf postDomFrontier) build(fn *Function) {
	for _, b := range fn.Blocks {
		if len(b.Succs) >= 2 {
			for _, s := range b.Succs {
				runner := s
				for runner != b.pdom.idom {
					rdf.add(runner, b)
					runner = runner.pdom.idom
				}
			}
		}
	}
}

func buildPostDomFrontier(fn *Function) postDomFrontier {
	rdf := make(postDomFrontier, len(fn.Blocks))
	rdf.build(fn)
	return rdf
}

func removeInstr(refs []Instruction, instr Instruction) []Instruction {
	i := 0
	for _, ref := range refs {
		if ref == instr {
			continue
		}
		refs[i] = ref
		i++
	}
	for j := i; j != len(refs); j++ {
		refs[j] = nil // aid GC
	}
	return refs[:i]
}

// lift replaces local and new Allocs accessed only with
// load/store by IR registers, inserting φ- and σ-nodes where necessary.
// The result is a program in pruned SSI form.
//
// Preconditions:
// - fn has no dead blocks (blockopt has run).
// - Def/use info (Operands and Referrers) is up-to-date.
// - The dominator tree is up-to-date.
//
func lift(fn *Function) {
	// TODO(adonovan): opt: lots of little optimizations may be
	// worthwhile here, especially if they cause us to avoid
	// buildDomFrontier.  For example:
	//
	// - Alloc never loaded?  Eliminate.
	// - Alloc never stored?  Replace all loads with a zero constant.
	// - Alloc stored once?  Replace loads with dominating store;
	//   don't forget that an Alloc is itself an effective store
	//   of zero.
	// - Alloc used only within a single block?
	//   Use degenerate algorithm avoiding φ-nodes.
	// - Consider synergy with scalar replacement of aggregates (SRA).
	//   e.g. *(&x.f) where x is an Alloc.
	//   Perhaps we'd get better results if we generated this as x.f
	//   i.e. Field(x, .f) instead of Load(FieldIndex(x, .f)).
	//   Unclear.
	//
	// But we will start with the simplest correct code.
	df := buildDomFrontier(fn)
	rdf := buildPostDomFrontier(fn)

	if debugLifting {
		title := false
		for i, blocks := range df {
			if blocks != nil {
				if !title {
					fmt.Fprintf(os.Stderr, "Dominance frontier of %s:\n", fn)
					title = true
				}
				fmt.Fprintf(os.Stderr, "\t%s: %s\n", fn.Blocks[i], blocks)
			}
		}
	}

	newPhis := make(newPhiMap, len(fn.Blocks))
	newSigmas := make(newSigmaMap, len(fn.Blocks))

	// During this pass we will replace some BasicBlock.Instrs
	// (allocs, loads and stores) with nil, keeping a count in
	// BasicBlock.gaps.  At the end we will reset Instrs to the
	// concatenation of all non-dead newPhis and non-nil Instrs
	// for the block, reusing the original array if space permits.

	// While we're here, we also eliminate 'rundefers'
	// instructions in functions that contain no 'defer'
	// instructions.
	usesDefer := false

	// Determine which allocs we can lift and number them densely.
	// The renaming phase uses this numbering for compact maps.
	numAllocs := 0
	for _, b := range fn.Blocks {
		b.gaps = 0
		b.rundefers = 0
		for _, instr := range b.Instrs {
			switch instr := instr.(type) {
			case *Alloc:
				index := -1
				if liftAlloc(df, rdf, instr, newPhis, newSigmas) {
					index = numAllocs
					numAllocs++
				}
				instr.index = index
			case *Defer:
				usesDefer = true
			case *RunDefers:
				b.rundefers++
			}
		}
	}

	// renaming maps an alloc (keyed by index) to its replacement
	// value.  Initially the renaming contains nil, signifying the
	// zero constant of the appropriate type; we construct the
	// Const lazily at most once on each path through the domtree.
	// TODO(adonovan): opt: cache per-function not per subtree.
	renaming := make([]Value, numAllocs)

	// Renaming.
	rename(fn.Blocks[0], renaming, newPhis, newSigmas)

	simplifyPhis(newPhis)

	// Eliminate dead φ- and σ-nodes.
	markLiveNodes(fn.Blocks, newPhis, newSigmas)

	// Prepend remaining live φ-nodes to each block and possibly kill rundefers.
	for _, b := range fn.Blocks {
		nps := newPhis[b.Index]
		head := make([]Instruction, 0, len(nps))
		for _, pred := range b.Preds {
			nss := newSigmas[pred.Index]
			idx := pred.succIndex(b)
			for _, newSigma := range nss {
				if sigma := newSigma.sigmas[idx]; sigma.live {
					head = append(head, sigma)

					// we didn't populate referrers before, as most
					// sigma nodes will be killed
					if refs := sigma.X.Referrers(); refs != nil {
						*refs = append(*refs, sigma)
					}
				} else {
					sigma.block = nil
				}
			}
		}
		for _, np := range nps {
			if np.phi.live {
				head = append(head, np.phi)
			} else {
				for _, edge := range np.phi.Edges {
					if refs := edge.Referrers(); refs != nil {
						*refs = removeInstr(*refs, np.phi)
					}
				}
				np.phi.block = nil
			}
		}

		rundefersToKill := b.rundefers
		if usesDefer {
			rundefersToKill = 0
		}

		j := len(head)
		if j+b.gaps+rundefersToKill == 0 {
			continue // fast path: no new phis or gaps
		}

		// Compact nps + non-nil Instrs into a new slice.
		dst := make([]Instruction, len(b.Instrs)+j-b.gaps-rundefersToKill)
		copy(dst, head)
		for _, instr := range b.Instrs {
			if instr == nil {
				continue
			}
			if !usesDefer {
				if _, ok := instr.(*RunDefers); ok {
					continue
				}
			}
			dst[j] = instr
			j++
		}
		b.Instrs = dst
	}

	// Remove any fn.Locals that were lifted.
	j := 0
	for _, l := range fn.Locals {
		if l.index < 0 {
			fn.Locals[j] = l
			j++
		}
	}
	// Nil out fn.Locals[j:] to aid GC.
	for i := j; i < len(fn.Locals); i++ {
		fn.Locals[i] = nil
	}
	fn.Locals = fn.Locals[:j]
}

func hasDirectReferrer(instr Instruction) bool {
	for _, instr := range *instr.Referrers() {
		switch instr.(type) {
		case *Phi, *Sigma:
			// ignore
		default:
			return true
		}
	}
	return false
}

func markLiveNodes(blocks []*BasicBlock, newPhis newPhiMap, newSigmas newSigmaMap) {
	// Phi and sigma nodes are considered live if a non-phi, non-sigma
	// node uses them. Once we find a node that is live, we mark all
	// of its operands as used, too.
	for _, npList := range newPhis {
		for _, np := range npList {
			phi := np.phi
			if !phi.live && hasDirectReferrer(phi) {
				markLivePhi(phi)
			}
		}
	}
	for _, npList := range newSigmas {
		for _, np := range npList {
			for _, sigma := range np.sigmas {
				if !sigma.live && hasDirectReferrer(sigma) {
					markLiveSigma(sigma)
				}
			}
		}
	}
	// Existing φ-nodes due to && and || operators
	// are all considered live (see Go issue 19622).
	for _, b := range blocks {
		for _, phi := range b.phis() {
			markLivePhi(phi.(*Phi))
		}
	}
}

func markLivePhi(phi *Phi) {
	phi.live = true
	for _, rand := range phi.Edges {
		switch rand := rand.(type) {
		case *Phi:
			if !rand.live {
				markLivePhi(rand)
			}
		case *Sigma:
			if !rand.live {
				markLiveSigma(rand)
			}
		}
	}
}

func markLiveSigma(sigma *Sigma) {
	sigma.live = true
	switch rand := sigma.X.(type) {
	case *Phi:
		if !rand.live {
			markLivePhi(rand)
		}
	case *Sigma:
		if !rand.live {
			markLiveSigma(rand)
		}
	}
}

// simplifyPhis replaces trivial phis with non-phi alternatives. Phi
// nodes where all edges are identical, or consist of only the phi
// itself and one other value, may be replaced with the value.
func simplifyPhis(newPhis newPhiMap) {
	// find all phis that are trivial and can be replaced with a
	// non-phi value. run until we reach a fixpoint, because replacing
	// a phi may make other phis trivial.
	for changed := true; changed; {
		changed = false
		for _, npList := range newPhis {
			for _, np := range npList {
				if np.phi.live {
					// we're reusing 'live' to mean 'dead' in the context of simplifyPhis
					continue
				}
				if r, ok := isUselessPhi(np.phi); ok {
					// useless phi, replace its uses with the
					// replacement value. the dead phi pass will clean
					// up the phi afterwards.
					replaceAll(np.phi, r)
					np.phi.live = true
					changed = true
				}
			}
		}
	}

	for _, npList := range newPhis {
		for _, np := range npList {
			np.phi.live = false
		}
	}
}

type BlockSet struct {
	idx    int
	values []bool
	count  int
}

func NewBlockSet(size int) *BlockSet {
	return &BlockSet{values: make([]bool, size)}
}

func (s *BlockSet) Set(s2 *BlockSet) {
	copy(s.values, s2.values)
	s.count = 0
	for _, v := range s.values {
		if v {
			s.count++
		}
	}
}

func (s *BlockSet) Num() int {
	return s.count
}

func (s *BlockSet) Has(b *BasicBlock) bool {
	if b.Index >= len(s.values) {
		return false
	}
	return s.values[b.Index]
}

// add adds b to the set and returns true if the set changed.
func (s *BlockSet) Add(b *BasicBlock) bool {
	if s.values[b.Index] {
		return false
	}
	s.count++
	s.values[b.Index] = true
	s.idx = b.Index

	return true
}

func (s *BlockSet) Clear() {
	for j := range s.values {
		s.values[j] = false
	}
	s.count = 0
}

// take removes an arbitrary element from a set s and
// returns its index, or returns -1 if empty.
func (s *BlockSet) Take() int {
	// [i, end]
	for i := s.idx; i < len(s.values); i++ {
		if s.values[i] {
			s.values[i] = false
			s.idx = i
			s.count--
			return i
		}
	}

	// [start, i)
	for i := 0; i < s.idx; i++ {
		if s.values[i] {
			s.values[i] = false
			s.idx = i
			s.count--
			return i
		}
	}

	return -1
}

// newPhi is a pair of a newly introduced φ-node and the lifted Alloc
// it replaces.
type newPhi struct {
	phi   *Phi
	alloc *Alloc
}

type newSigma struct {
	alloc  *Alloc
	sigmas []*Sigma
}

// newPhiMap records for each basic block, the set of newPhis that
// must be prepended to the block.
type newPhiMap [][]newPhi
type newSigmaMap [][]newSigma

// liftAlloc determines whether alloc can be lifted into registers,
// and if so, it populates newPhis with all the φ-nodes it may require
// and returns true.
func liftAlloc(df domFrontier, rdf postDomFrontier, alloc *Alloc, newPhis newPhiMap, newSigmas newSigmaMap) bool {
	fn := alloc.Parent()

	defblocks := fn.blockset(0)
	useblocks := fn.blockset(1)
	Aphi := fn.blockset(2)
	Asigma := fn.blockset(3)
	W := fn.blockset(4)

	// Don't lift aggregates into registers, because we don't have
	// a way to express their zero-constants.
	switch deref(alloc.Type()).Underlying().(type) {
	case *types.Array, *types.Struct:
		return false
	}

	// Don't lift named return values in functions that defer
	// calls that may recover from panic.
	if fn.hasDefer {
		for _, nr := range fn.namedResults {
			if nr == alloc {
				return false
			}
		}
	}

	// Compute defblocks, the set of blocks containing a
	// definition of the alloc cell.
	for _, instr := range *alloc.Referrers() {
		// Bail out if we discover the alloc is not liftable;
		// the only operations permitted to use the alloc are
		// loads/stores into the cell, and DebugRef.
		switch instr := instr.(type) {
		case *Store:
			if instr.Val == alloc {
				return false // address used as value
			}
			if instr.Addr != alloc {
				panic("Alloc.Referrers is inconsistent")
			}
			defblocks.Add(instr.Block())
		case *Load:
			if instr.X != alloc {
				panic("Alloc.Referrers is inconsistent")
			}
			useblocks.Add(instr.Block())
			for _, ref := range *instr.Referrers() {
				useblocks.Add(ref.Block())
			}
		case *DebugRef:
			// ok
		default:
			return false // some other instruction
		}
	}
	// The Alloc itself counts as a (zero) definition of the cell.
	defblocks.Add(alloc.Block())

	if debugLifting {
		fmt.Fprintln(os.Stderr, "\tlifting ", alloc, alloc.Name())
	}

	// Φ-insertion.
	//
	// What follows is the body of the main loop of the insert-φ
	// function described by Cytron et al, but instead of using
	// counter tricks, we just reset the 'hasAlready' and 'work'
	// sets each iteration.  These are bitmaps so it's pretty cheap.

	// Initialize W and work to defblocks.

	for change := true; change; {
		change = false
		{
			// Traverse iterated dominance frontier, inserting φ-nodes.
			W.Set(defblocks)

			for i := W.Take(); i != -1; i = W.Take() {
				n := fn.Blocks[i]
				for _, y := range df[n.Index] {
					if Aphi.Add(y) {
						// Create φ-node.
						// It will be prepended to v.Instrs later, if needed.
						phi := &Phi{
							Edges:   make([]Value, len(y.Preds)),
							Comment: alloc.Comment,
						}

						phi.source = alloc.source
						phi.setType(deref(alloc.Type()))
						phi.block = y
						if debugLifting {
							fmt.Fprintf(os.Stderr, "\tplace %s = %s at block %s\n", phi.Name(), phi, y)
						}
						newPhis[y.Index] = append(newPhis[y.Index], newPhi{phi, alloc})

						for _, p := range y.Preds {
							useblocks.Add(p)
						}
						change = true
						if defblocks.Add(y) {
							W.Add(y)
						}
					}
				}
			}
		}

		{
			W.Set(useblocks)
			for i := W.Take(); i != -1; i = W.Take() {
				n := fn.Blocks[i]
				for _, y := range rdf[n.Index] {
					// OPT(dh): if we had liveness information, we
					// could avoid adding sigma nodes for already dead
					// variables. but would calculating liveness
					// information be cheaper than pruning dead sigmas
					// later?
					if Asigma.Add(y) {
						sigmas := make([]*Sigma, 0, len(y.Succs))
						for _, succ := range y.Succs {
							sigma := &Sigma{
								From:    y,
								X:       alloc,
								Comment: alloc.Comment,
							}
							sigma.source = alloc.source
							sigma.setType(deref(alloc.Type()))
							sigma.block = succ
							sigmas = append(sigmas, sigma)
						}

						newSigmas[y.Index] = append(newSigmas[y.Index], newSigma{alloc, sigmas})
						for _, s := range y.Succs {
							defblocks.Add(s)
						}
						change = true
						if useblocks.Add(y) {
							W.Add(y)
						}
					}
				}
			}
		}
	}

	return true
}

// replaceAll replaces all intraprocedural uses of x with y,
// updating x.Referrers and y.Referrers.
// Precondition: x.Referrers() != nil, i.e. x must be local to some function.
//
func replaceAll(x, y Value) {
	var rands []*Value
	pxrefs := x.Referrers()
	pyrefs := y.Referrers()
	for _, instr := range *pxrefs {
		rands = instr.Operands(rands[:0]) // recycle storage
		for _, rand := range rands {
			if *rand != nil {
				if *rand == x {
					*rand = y
				}
			}
		}
		if pyrefs != nil {
			*pyrefs = append(*pyrefs, instr) // dups ok
		}
	}
	*pxrefs = nil // x is now unreferenced
}

// renamed returns the value to which alloc is being renamed,
// constructing it lazily if it's the implicit zero initialization.
//
func renamed(fn *Function, renaming []Value, alloc *Alloc) Value {
	v := renaming[alloc.index]
	if v == nil {
		v = emitConst(fn, zeroConst(deref(alloc.Type())))
		renaming[alloc.index] = v
	}
	return v
}

// rename implements the Cytron et al-based SSI renaming algorithm, a
// preorder traversal of the dominator tree replacing all loads of
// Alloc cells with the value stored to that cell by the dominating
// store instruction.
//
// renaming is a map from *Alloc (keyed by index number) to its
// dominating stored value; newPhis[x] is the set of new φ-nodes to be
// prepended to block x.
//
func rename(u *BasicBlock, renaming []Value, newPhis newPhiMap, newSigmas newSigmaMap) {
	// Each φ-node becomes the new name for its associated Alloc.
	for _, np := range newPhis[u.Index] {
		phi := np.phi
		alloc := np.alloc
		renaming[alloc.index] = phi
	}

	// Rename loads and stores of allocs.
	for i, instr := range u.Instrs {
		switch instr := instr.(type) {
		case *Alloc:
			if instr.index >= 0 { // store of zero to Alloc cell
				// Replace dominated loads by the zero value.
				renaming[instr.index] = nil
				if debugLifting {
					fmt.Fprintf(os.Stderr, "\tkill alloc %s\n", instr)
				}
				// Delete the Alloc.
				u.Instrs[i] = nil
				u.gaps++
			}

		case *Store:
			if alloc, ok := instr.Addr.(*Alloc); ok && alloc.index >= 0 { // store to Alloc cell
				// Replace dominated loads by the stored value.
				renaming[alloc.index] = instr.Val
				if debugLifting {
					fmt.Fprintf(os.Stderr, "\tkill store %s; new value: %s\n",
						instr, instr.Val.Name())
				}
				if refs := instr.Addr.Referrers(); refs != nil {
					*refs = removeInstr(*refs, instr)
				}
				if refs := instr.Val.Referrers(); refs != nil {
					*refs = removeInstr(*refs, instr)
				}
				// Delete the Store.
				u.Instrs[i] = nil
				u.gaps++
			}

		case *Load:
			if alloc, ok := instr.X.(*Alloc); ok && alloc.index >= 0 { // load of Alloc cell
				// In theory, we wouldn't be able to replace loads
				// directly, because a loaded value could be used in
				// different branches, in which case it should be
				// replaced with different sigma nodes. But we can't
				// simply defer replacement, either, because then
				// later stores might incorrectly affect this load.
				//
				// To avoid doing renaming on _all_ values (instead of
				// just loads and stores like we're doing), we make
				// sure during code generation that each load is only
				// used in one block. For example, in constant switch
				// statements, where the tag is only evaluated once,
				// we store it in a temporary and load it for each
				// comparison, so that we have individual loads to
				// replace.
				newval := renamed(u.Parent(), renaming, alloc)
				if debugLifting {
					fmt.Fprintf(os.Stderr, "\tupdate load %s = %s with %s\n",
						instr.Name(), instr, newval)
				}
				replaceAll(instr, newval)
				u.Instrs[i] = nil
				u.gaps++
			}

		case *DebugRef:
			if x, ok := instr.X.(*Alloc); ok && x.index >= 0 {
				if instr.IsAddr {
					instr.X = renamed(u.Parent(), renaming, x)
					instr.IsAddr = false

					// Add DebugRef to instr.X's referrers.
					if refs := instr.X.Referrers(); refs != nil {
						*refs = append(*refs, instr)
					}
				} else {
					// A source expression denotes the address
					// of an Alloc that was optimized away.
					instr.X = nil

					// Delete the DebugRef.
					u.Instrs[i] = nil
					u.gaps++
				}
			}
		}
	}

	// update all outgoing sigma nodes with the dominating store
	for _, sigmas := range newSigmas[u.Index] {
		for _, sigma := range sigmas.sigmas {
			sigma.X = renamed(u.Parent(), renaming, sigmas.alloc)
		}
	}

	// For each φ-node in a CFG successor, rename the edge.
	for succi, v := range u.Succs {
		phis := newPhis[v.Index]
		if len(phis) == 0 {
			continue
		}
		i := v.predIndex(u)
		for _, np := range phis {
			phi := np.phi
			alloc := np.alloc
			// if there's a sigma node, use it, else use the dominating value
			var newval Value
			for _, sigmas := range newSigmas[u.Index] {
				if sigmas.alloc == alloc {
					newval = sigmas.sigmas[succi]
					break
				}
			}
			if newval == nil {
				newval = renamed(u.Parent(), renaming, alloc)
			}
			if debugLifting {
				fmt.Fprintf(os.Stderr, "\tsetphi %s edge %s -> %s (#%d) (alloc=%s) := %s\n",
					phi.Name(), u, v, i, alloc.Name(), newval.Name())
			}
			phi.Edges[i] = newval
			if prefs := newval.Referrers(); prefs != nil {
				*prefs = append(*prefs, phi)
			}
		}
	}

	// Continue depth-first recursion over domtree, pushing a
	// fresh copy of the renaming map for each subtree.
	r := make([]Value, len(renaming))
	for _, v := range u.dom.children {
		// XXX add debugging
		copy(r, renaming)

		// on entry to a block, the incoming sigma nodes become the new values for their alloc
		if idx := u.succIndex(v); idx != -1 {
			for _, sigma := range newSigmas[u.Index] {
				r[sigma.alloc.index] = sigma.sigmas[idx]
			}
		}
		rename(v, r, newPhis, newSigmas)
	}

}
