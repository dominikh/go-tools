// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ir

import (
	"go/types"

	"honnef.co/go/tools/go/types/typeutil"
)

// Utilities for dealing with type sets.

const debug = false

// typeset is an iterator over the (type/underlying type) pairs of the
// specific type terms of the type set implied by t.
// If t is a type parameter, the implied type set is the type set of t's constraint.
// In that case, if there are no specific terms, typeset calls yield with (nil, nil).
// If t is not a type parameter, the implied type set consists of just t.
// In any case, typeset is guaranteed to call yield at least once.
func typeset(typ types.Type, yield func(t, u types.Type) bool) {
	switch typ := types.Unalias(typ).(type) {
	case *types.TypeParam, *types.Interface:
		terms := termListOf(typ)
		if len(terms) == 0 {
			yield(nil, nil)
			return
		}
		for _, term := range terms {
			u := types.Unalias(term.Type())
			if !term.Tilde() {
				u = u.Underlying()
			}
			if debug {
				assert(types.Identical(u, u.Underlying()))
			}
			if !yield(term.Type(), u) {
				break
			}
		}
		return
	default:
		yield(typ, typ.Underlying())
	}
}

// termListOf returns the type set of typ as a normalized term set. Returns an empty set on an error.
func termListOf(typ types.Type) []*types.Term {
	return typeutil.NewTypeSet(typ).Terms
}

// underIs calls f with the underlying types of the type terms
// of the type set of typ and reports whether all calls to f returned true.
// If there are no specific terms, underIs returns the result of f(nil).
func underIs(typ types.Type, f func(types.Type) bool) bool {
	var ok bool
	typeset(typ, func(t, u types.Type) bool {
		ok = f(u)
		return ok
	})
	return ok
}
