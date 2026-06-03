// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dense_test

import (
	"honnef.co/go/tools/analysis/dfa"
)

type nodeSet uint64

type nodeSetUnion struct{}

func (nodeSetUnion) Ident() nodeSet             { return 0 }
func (nodeSetUnion) Equals(a, b nodeSet) bool   { return a == b }
func (nodeSetUnion) Merge(a, b nodeSet) nodeSet { return a | b }

var _ dfa.Semilattice[nodeSet] = nodeSetUnion{}

func set(nodes ...int) nodeSet {
	var out nodeSet
	for _, node := range nodes {
		if node < 0 || node >= 64 {
			panic("NodeID out of range")
		}
		out |= 1 << node
	}
	return out
}
