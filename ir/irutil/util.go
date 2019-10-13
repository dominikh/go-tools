package irutil

import (
	"honnef.co/go/tools/ir"
)

func Reachable(from, to *ir.BasicBlock) bool {
	if from == to {
		return true
	}
	if from.Dominates(to) {
		return true
	}

	found := false
	Walk(from, func(b *ir.BasicBlock) bool {
		if b == to {
			found = true
			return false
		}
		return true
	})
	return found
}

func Walk(b *ir.BasicBlock, fn func(*ir.BasicBlock) bool) {
	seen := map[*ir.BasicBlock]bool{}
	wl := []*ir.BasicBlock{b}
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

func Vararg(x *ir.Slice) ([]ir.Value, bool) {
	var out []ir.Value
	slice, ok := x.X.(*ir.Alloc)
	if !ok || slice.Comment != "varargs" {
		return nil, false
	}
	for _, ref := range *slice.Referrers() {
		idx, ok := ref.(*ir.IndexAddr)
		if !ok {
			continue
		}
		v := (*idx.Referrers())[0].(*ir.Store).Val
		out = append(out, v)
	}
	return out, true
}
