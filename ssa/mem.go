package ssa

import (
	"fmt"
	"go/token"
	"go/types"
)

var tMemory = types.NewNamed(types.NewTypeName(token.NoPos, nil, "Memory", nil), types.Typ[types.UnsafePointer], nil)

type InitMem struct {
	register
}

func (*InitMem) Operands(rands []*Value) []*Value {
	return rands
}

func (*InitMem) String() string {
	return "InitMem"
}

type ReturnValues struct {
	register
	Mem Value
}

func (retv *ReturnValues) Operands(rands []*Value) []*Value {
	return append(rands, &retv.Mem)
}

func (retv *ReturnValues) String() string {
	return fmt.Sprintf("ReturnValues %s", relName(retv.Mem, retv))
}
