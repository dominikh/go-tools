package ssautil

import (
	"honnef.co/go/tools/ssa"
)

func Reachable(from, to *ssa.BasicBlock) bool {
	if from == to {
		return true
	}
	if from.Dominates(to) {
		return true
	}

	found := false
	Walk(from, func(b *ssa.BasicBlock) bool {
		if b == to {
			found = true
			return false
		}
		return true
	})
	return found
}

func Walk(b *ssa.BasicBlock, fn func(*ssa.BasicBlock) bool) {
	seen := map[*ssa.BasicBlock]bool{}
	wl := []*ssa.BasicBlock{b}
	for len(wl) > 0 {
		b := wl[len(wl)-1]
		wl = wl[:len(wl)-1]
		if seen[b] {
			continue
		}
		seen[b] = true
		if !fn(b) {
			continue
		}
		wl = append(wl, b.Succs...)
	}
}

func Vararg(x *ssa.Slice) ([]ssa.Value, bool) {
	var out []ssa.Value
	slice, ok := x.X.(*ssa.Alloc)
	if !ok || slice.Comment != "varargs" {
		return nil, false
	}
	for _, ref := range *slice.Referrers() {
		idx, ok := ref.(*ssa.IndexAddr)
		if !ok {
			continue
		}
		v := (*idx.Referrers())[0].(*ssa.Store).Val
		out = append(out, v)
	}
	return out, true
}

func IsCallResult(v ssa.Value) (*ssa.Call, bool) {
	ex, ok := v.(*ssa.Extract)
	if !ok {
		return nil, false
	}
	retv, ok := ex.Tuple.(*ssa.ReturnValues)
	if !ok {
		return nil, false
	}
	call, ok := retv.Mem.(*ssa.Call)
	return call, ok
}

func CallResult(v *ssa.Call) []*ssa.Extract {
	var out []*ssa.Extract
	if v.Common().Signature().Results().Len() != 1 {
		return nil
	}
	refs := v.Referrers()
	if refs == nil {
		return nil
	}
	for _, ref := range *refs {
		if retv, ok := ref.(*ssa.ReturnValues); ok {
			refs2 := retv.Referrers()
			if refs2 == nil {
				continue
			}
			for _, ref2 := range *refs2 {
				if ex, ok := ref2.(*ssa.Extract); ok {
					out = append(out, ex)
				}
			}
		}
	}
	return out
}
