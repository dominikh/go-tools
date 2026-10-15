package main

import (
	"go/build"
	"go/types"
	"testing"

	"honnef.co/go/tools/go/gcsizes"
)

func TestSizesIncludesNestedTrailingPadding(t *testing.T) {
	byteField := func(name string, typ types.Type) *types.Var {
		return types.NewField(0, nil, name, typ, false)
	}
	inner := types.NewStruct([]*types.Var{
		byteField("A", types.Typ[types.Byte]),
		byteField("B", types.NewArray(types.Typ[types.Int64], 0)),
	}, nil)
	outer := types.NewStruct([]*types.Var{
		byteField("X", types.Typ[types.Byte]),
		byteField("I", inner),
		byteField("Z", types.Typ[types.Byte]),
	}, nil)

	sizesForArch := gcsizes.ForArch(build.Default.GOARCH)
	outerOffsets := sizesForArch.Offsetsof([]*types.Var{outer.Field(0), outer.Field(1), outer.Field(2)})
	innerOffsets := sizesForArch.Offsetsof([]*types.Var{inner.Field(0), inner.Field(1)})
	innerStart := outerOffsets[1]
	lastFieldSize := sizesForArch.Sizeof(inner.Field(1).Type())
	if lastFieldSize == 0 {
		lastFieldSize = 1
	}
	wantStart := innerStart + innerOffsets[1] + lastFieldSize
	wantEnd := innerStart + sizesForArch.Sizeof(inner)

	fields := sizes(outer, "S", 0, nil)
	for _, field := range fields {
		if field.IsPadding && field.Start == wantStart && field.End == wantEnd {
			return
		}
	}
	t.Fatalf("sizes(%s) omitted nested trailing padding [%d,%d): %#v", outer, wantStart, wantEnd, fields)
}
