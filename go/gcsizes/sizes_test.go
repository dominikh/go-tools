package gcsizes

import (
	"go/types"
	"testing"
)

func TestForArch(t *testing.T) {
	tests := []struct {
		arch     string
		wordSize int64
		maxAlign int64
	}{
		{arch: "386", wordSize: 4, maxAlign: 4},
		{arch: "arm", wordSize: 4, maxAlign: 4},
		{arch: "amd64p32", wordSize: 4, maxAlign: 8},
		{arch: "amd64", wordSize: 8, maxAlign: 8},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			sizes := ForArch(tt.arch)
			if sizes.WordSize != tt.wordSize || sizes.MaxAlign != tt.maxAlign {
				t.Fatalf("ForArch(%q) = {%d, %d}, want {%d, %d}", tt.arch, sizes.WordSize, sizes.MaxAlign, tt.wordSize, tt.maxAlign)
			}
			if got := sizes.Sizeof(types.Typ[types.Int]); got != tt.wordSize {
				t.Fatalf("Sizeof(int) = %d, want %d", got, tt.wordSize)
			}
		})
	}
}
