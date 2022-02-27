package typeutil

import (
	"go/types"

	"golang.org/x/exp/typeparams"
)

func All(terms []*typeparams.Term, fn func(*typeparams.Term) bool) bool {
	if len(terms) == 0 {
		return fn(nil)
	}
	for _, term := range terms {
		if !fn(term) {
			return false
		}
	}
	return true
}

func Any(terms []*typeparams.Term, fn func(*typeparams.Term) bool) bool {
	if len(terms) == 0 {
		return fn(nil)
	}
	for _, term := range terms {
		if fn(term) {
			return true
		}
	}
	return false
}

func AllAndAny(terms []*typeparams.Term, fn func(*typeparams.Term) bool) bool {
	return All(terms, func(term *typeparams.Term) bool {
		if term == nil {
			return false
		}
		return fn(term)
	})
}

func CoreType(t types.Type) types.Type {
	if t, ok := t.(*typeparams.TypeParam); ok {
		terms, err := typeparams.NormalTerms(t)
		if err != nil || len(terms) == 0 {
			return nil
		}
		typ := terms[0].Type().Underlying()
		for _, term := range terms[1:] {
			ut := term.Type().Underlying()
			if types.Identical(typ, ut) {
				continue
			}

			ch1, ok := typ.(*types.Chan)
			if !ok {
				return nil
			}
			ch2, ok := ut.(*types.Chan)
			if !ok {
				return nil
			}
			if ch1.Dir() == types.SendRecv {
				// typ is currently a bidirectional channel. The term's type is either also bidirectional, or
				// unidirectional. Use the term's type.
				typ = ut
			} else if ch1.Dir() != ch2.Dir() {
				// typ is not bidirectional and typ and term disagree about the direction
				return nil
			}
		}
		return typ
	}
	return t
}
