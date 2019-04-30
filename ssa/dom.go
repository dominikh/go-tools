// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa

// This file defines algorithms related to dominance.

// Dominator tree construction ----------------------------------------
//
// We use the algorithm described in Lengauer & Tarjan. 1979.  A fast
// algorithm for finding dominators in a flowgraph.
// http://doi.acm.org/10.1145/357062.357071
//
// We also apply the optimizations to SLT described in Georgiadis et
// al, Finding Dominators in Practice, JGAA 2006,
// http://jgaa.info/accepted/2006/GeorgiadisTarjanWerneck2006.10.1.pdf
// to avoid the need for buckets of size > 1.

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"sort"
)

// Idom returns the block that immediately dominates b:
// its parent in the dominator tree, if any.
// The entry node (b.Index==0) does not have a parent.
//
func (b *BasicBlock) Idom() *BasicBlock { return b.dom.idom }

// Dominees returns the list of blocks that b immediately dominates:
// its children in the dominator tree.
//
func (b *BasicBlock) Dominees() []*BasicBlock { return b.dom.children }

// Dominates reports whether b dominates c.
func (b *BasicBlock) Dominates(c *BasicBlock) bool {
	return b.dom.pre <= c.dom.pre && c.dom.post <= b.dom.post
}

type byDomPreorder []*BasicBlock

func (a byDomPreorder) Len() int           { return len(a) }
func (a byDomPreorder) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDomPreorder) Less(i, j int) bool { return a[i].dom.pre < a[j].dom.pre }

// DomPreorder returns a new slice containing the blocks of f in
// dominator tree preorder.
//
func (f *Function) DomPreorder() []*BasicBlock {
	n := len(f.Blocks)
	order := make(byDomPreorder, n)
	copy(order, f.Blocks)
	sort.Sort(order)
	return order
}

// domInfo contains a BasicBlock's dominance information.
type domInfo struct {
	idom      *BasicBlock   // immediate dominator (parent in domtree)
	children  []*BasicBlock // nodes immediately dominated by this one
	pre, post int32         // pre- and post-order numbering within domtree
}

// buildDomTree computes the dominator tree of f using the LT algorithm.
// Precondition: all blocks are reachable (e.g. optimizeBlocks has been run).
//
func buildDomTree(fn *Function) {
	// The step numbers refer to the original LT paper; the
	// reordering is due to Georgiadis.

	// Clear any previous domInfo.
	for _, b := range fn.Blocks {
		b.dom = domInfo{}
	}

	idoms := make([]*BasicBlock, len(fn.Blocks))

	order := make([]*BasicBlock, 0, len(fn.Blocks))
	var seen BlockSet
	var dfs func(b *BasicBlock)
	dfs = func(b *BasicBlock) {
		if !seen.Add(b) {
			return
		}
		for _, succ := range b.Succs {
			dfs(succ)
		}
		order = append(order, b)
		b.post = len(order) - 1
	}
	dfs(fn.Blocks[0])

	for i := 0; i < len(order)/2; i++ {
		o := len(order) - i - 1
		order[i], order[o] = order[o], order[i]
	}

	idoms[fn.Blocks[0].Index] = fn.Blocks[0]
	changed := true
	for changed {
		changed = false
		// iterate over all nodes in reverse postorder, except for the
		// entry node
		for _, b := range order[1:] {
			var newIdom *BasicBlock
			for _, p := range b.Preds {
				if idoms[p.Index] == nil {
					continue
				}
				if newIdom == nil {
					newIdom = p
				} else {
					finger1 := p
					finger2 := newIdom
					for finger1 != finger2 {
						for finger1.post < finger2.post {
							finger1 = idoms[finger1.Index]
						}
						for finger2.post < finger1.post {
							finger2 = idoms[finger2.Index]
						}
					}
					newIdom = finger1
				}
			}

			if idoms[b.Index] != newIdom {
				idoms[b.Index] = newIdom
				changed = true
			}
		}
	}

	for i, b := range idoms {
		fn.Blocks[i].dom.idom = b
		if i == b.Index {
			continue
		}
		b.dom.children = append(b.dom.children, fn.Blocks[i])
	}

	numberDomTree(fn.Blocks[0], 0, 0)

	// printDomTreeDot(os.Stderr, f) // debugging
	// printDomTreeText(os.Stderr, root, 0) // debugging

	if fn.Prog.mode&SanityCheckFunctions != 0 {
		sanityCheckDomTree(fn)
	}
}

// numberDomTree sets the pre- and post-order numbers of a depth-first
// traversal of the dominator tree rooted at v.  These are used to
// answer dominance queries in constant time.
//
func numberDomTree(v *BasicBlock, pre, post int32) (int32, int32) {
	v.dom.pre = pre
	pre++
	for _, child := range v.dom.children {
		pre, post = numberDomTree(child, pre, post)
	}
	v.dom.post = post
	post++
	return pre, post
}

// Testing utilities ----------------------------------------

// sanityCheckDomTree checks the correctness of the dominator tree
// computed by the LT algorithm by comparing against the dominance
// relation computed by a naive Kildall-style forward dataflow
// analysis (Algorithm 10.16 from the "Dragon" book).
//
func sanityCheckDomTree(f *Function) {
	n := len(f.Blocks)

	// D[i] is the set of blocks that dominate f.Blocks[i],
	// represented as a bit-set of block indices.
	D := make([]big.Int, n)

	one := big.NewInt(1)

	// all is the set of all blocks; constant.
	var all big.Int
	all.Set(one).Lsh(&all, uint(n)).Sub(&all, one)

	// Initialization.
	for i := range f.Blocks {
		if i == 0 {
			// A root is dominated only by itself.
			D[i].SetBit(&D[0], 0, 1)
		} else {
			// All other blocks are (initially) dominated
			// by every block.
			D[i].Set(&all)
		}
	}

	// Iteration until fixed point.
	for changed := true; changed; {
		changed = false
		for i, b := range f.Blocks {
			if i == 0 {
				continue
			}
			// Compute intersection across predecessors.
			var x big.Int
			x.Set(&all)
			for _, pred := range b.Preds {
				x.And(&x, &D[pred.Index])
			}
			x.SetBit(&x, i, 1) // a block always dominates itself.
			if D[i].Cmp(&x) != 0 {
				D[i].Set(&x)
				changed = true
			}
		}
	}

	// Check the entire relation.  O(n^2).
	ok := true
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			b, c := f.Blocks[i], f.Blocks[j]
			actual := b.Dominates(c)
			expected := D[j].Bit(i) == 1
			if actual != expected {
				fmt.Fprintf(os.Stderr, "dominates(%s, %s)==%t, want %t\n", b, c, actual, expected)
				ok = false
			}
		}
	}

	preorder := f.DomPreorder()
	for _, b := range f.Blocks {
		if got := preorder[b.dom.pre]; got != b {
			fmt.Fprintf(os.Stderr, "preorder[%d]==%s, want %s\n", b.dom.pre, got, b)
			ok = false
		}
	}

	if !ok {
		panic("sanityCheckDomTree failed for " + f.String())
	}

}

// Printing functions ----------------------------------------

// printDomTree prints the dominator tree as text, using indentation.
//lint:ignore U1000 used during debugging
func printDomTreeText(buf *bytes.Buffer, v *BasicBlock, indent int) {
	fmt.Fprintf(buf, "%*s%s\n", 4*indent, "", v)
	for _, child := range v.dom.children {
		printDomTreeText(buf, child, indent+1)
	}
}

// printDomTreeDot prints the dominator tree of f in AT&T GraphViz
// (.dot) format.
//lint:ignore U1000 used during debugging
func printDomTreeDot(buf *bytes.Buffer, f *Function) {
	fmt.Fprintln(buf, "//", f)
	fmt.Fprintln(buf, "digraph domtree {")
	for i, b := range f.Blocks {
		v := b.dom
		fmt.Fprintf(buf, "\tn%d [label=\"%s (%d, %d)\",shape=\"rectangle\"];\n", v.pre, b, v.pre, v.post)
		// TODO(adonovan): improve appearance of edges
		// belonging to both dominator tree and CFG.

		// Dominator tree edge.
		if i != 0 {
			fmt.Fprintf(buf, "\tn%d -> n%d [style=\"solid\",weight=100];\n", v.idom.dom.pre, v.pre)
		}
		// CFG edges.
		for _, pred := range b.Preds {
			fmt.Fprintf(buf, "\tn%d -> n%d [style=\"dotted\",weight=0];\n", pred.dom.pre, v.pre)
		}
	}
	fmt.Fprintln(buf, "}")
}
