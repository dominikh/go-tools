package functions

import (
	"go/types"

	"honnef.co/go/tools/ssa"
)

func concreteReturnTypes(fn *ssa.Function) []types.Type {
	// TODO(dh): support functions with >1 return values
	if fn.Signature.Results().Len() != 1 {
		return nil
	}
	typ := fn.Signature.Results().At(0).Type()
	if _, ok := typ.Underlying().(*types.Interface); !ok {
		return nil
	}
	var typs []types.Type
	for _, block := range fn.Blocks {
		if len(block.Instrs) == 0 {
			continue
		}
		ret, ok := block.Instrs[len(block.Instrs)-1].(*ssa.Return)
		if !ok {
			continue
		}
		if ins, ok := ret.Results[0].(*ssa.MakeInterface); ok {
			typs = append(typs, ins.X.Type())
		} else {
			return nil
		}
	}
	// TODO(dh): deduplicate typs
	return typs
}
