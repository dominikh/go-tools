package ir

import (
	"fmt"
	"testing"
)

func TestReplaceAllRepeatedCompositeValue(t *testing.T) {
	const size = 32

	x := &Alloc{}
	y := &Alloc{}
	instr := &CompositeValue{Values: make([]Value, size)}
	for i := range instr.Values {
		instr.Values[i] = x
		x.referrers = append(x.referrers, instr)
	}

	replaceAll(x, y)

	for i, value := range instr.Values {
		if value != y {
			t.Fatalf("operand %d: got %p, want %p", i, value, y)
		}
	}
	if len(x.referrers) != 0 {
		t.Fatalf("old referrers: got %d, want 0", len(x.referrers))
	}
	if len(y.referrers) != size {
		t.Fatalf("new referrers: got %d, want %d", len(y.referrers), size)
	}
}

func BenchmarkReplaceAllRepeatedCompositeValue(b *testing.B) {
	for _, size := range []int{16, 64, 256, 1024} {
		b.Run(fmt.Sprintf("operands=%d", size), func(b *testing.B) {
			x := &Alloc{}
			y := &Alloc{}
			instr := &CompositeValue{Values: make([]Value, size)}
			refs := make([]Instruction, size)
			for i := range refs {
				refs[i] = instr
			}
			x.referrers = refs
			for i := range instr.Values {
				instr.Values[i] = x
			}

			b.ReportAllocs()
			for b.Loop() {
				replaceAll(x, y)
				x, y = y, x
			}
		})
	}
}
