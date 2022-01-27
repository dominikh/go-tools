package vrp

import (
	"fmt"
	"go/constant"
	"go/types"

	"honnef.co/go/tools/go/ir"
)

func NewInt[T int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64](n T) Numeric {
	switch n := interface{}(n).(type) {
	case int8:
		return Int[int8]{n}
	case int16:
		return Int[int16]{n}
	case int32:
		return Int[int32]{n}
	case int64:
		return Int[int64]{n}
	case uint8:
		return Uint[uint8]{n}
	case uint16:
		return Uint[uint16]{n}
	case uint32:
		return Uint[uint32]{n}
	case uint64:
		return Uint[uint64]{n}
	default:
		panic("unreachable")
	}
}

type Int[T int8 | int16 | int32 | int64] struct {
	v T
}

func (n Int[T]) Add(o Numeric) (Numeric, bool) {
	switch o := o.(type) {
	case Infinity:
		if o.negative {
			panic("x + -∞ is not defined")
		}
		return o, false
	case Int[T]:
		r := n.v + o.v
		of := (r > n.v) != (o.v > 0)
		return Int[T]{r}, of
	default:
		panic(fmt.Sprintf("incompatible types %T and %T", n, o))
	}
}

func (n Int[T]) Sub(o Numeric) (Numeric, bool) {
	switch o := o.(type) {
	case Infinity:
		if o.negative {
			return Inf, false
		} else {
			return NegInf, false
		}
	case Int[T]:
		r := n.v - o.v
		of := (r < n.v) != (o.v > 0)
		return Int[T]{v: r}, of
	default:
		panic(fmt.Sprintf("incompatible types %T and %T", n, o))
	}
}

func (n Int[T]) Cmp(o Numeric) int {
	switch o := o.(type) {
	case Infinity:
		if o.negative {
			return 1
		} else {
			return -1
		}
	case Int[T]:
		if n.v > o.v {
			return 1
		} else if n.v == o.v {
			return 0
		} else {
			return -1
		}
	default:
		panic(fmt.Sprintf("incompatible types %T and %T", n, o))
	}
}

func (n Int[T]) Dec() (Numeric, bool) { return n.Sub(Int[T]{1}) }
func (n Int[T]) Inc() (Numeric, bool) { return n.Add(Int[T]{1}) }
func (n Int[T]) Negative() bool       { return n.v < 0 }
func (n Int[T]) String() string       { return fmt.Sprintf("%d", n.v) }

type Uint[T uint8 | uint16 | uint32 | uint64] struct {
	v T
}

func (n Uint[T]) Add(o Numeric) (Numeric, bool) {
	switch o := o.(type) {
	case Infinity:
		if o.negative {
			panic("x + -∞ is not defined")
		}
		return o, false
	case Uint[T]:
		r := n.v + o.v
		of := r < n.v
		return Uint[T]{v: r}, of
	default:
		panic(fmt.Sprintf("incompatible types %T and %T", n, o))
	}
}

func (n Uint[T]) Sub(o Numeric) (Numeric, bool) {
	switch o := o.(type) {
	case Infinity:
		if o.negative {
			return Inf, false
		} else {
			return NegInf, false
		}
	case Uint[T]:
		r := n.v - o.v
		of := r > n.v
		return Uint[T]{v: r}, of
	default:
		panic(fmt.Sprintf("incompatible types %T and %T", n, o))
	}
}

func (n Uint[T]) Cmp(o Numeric) int {
	switch o := o.(type) {
	case Infinity:
		if o.negative {
			return 1
		} else {
			return -1
		}
	case Uint[T]:
		if n.v > o.v {
			return 1
		} else if n.v == o.v {
			return 0
		} else {
			return -1
		}
	default:
		panic(fmt.Sprintf("incompatible types %T and %T", n, o))
	}
}

func (n Uint[T]) Dec() (Numeric, bool) { return n.Sub(Uint[T]{1}) }
func (n Uint[T]) Inc() (Numeric, bool) { return n.Add(Uint[T]{1}) }
func (n Uint[T]) Negative() bool       { return n.v < 0 }
func (n Uint[T]) String() string       { return fmt.Sprintf("%d", n.v) }

func ConstToNumeric(k *ir.Const) Numeric {
	typ := k.Type().Underlying().(*types.Basic)
	// XXX don't assume 64 bit
	std := types.StdSizes{WordSize: 8, MaxAlign: 1}
	if (typ.Info() & types.IsUnsigned) == 0 {
		n, exact := constant.Int64Val(constant.ToInt(k.Value))
		if !exact {
			panic("cannot represent constant")
		}
		width := int(std.Sizeof(typ)) * 8
		switch width {
		case 8:
			return Int[int8]{int8(n)}
		case 16:
			return Int[int16]{int16(n)}
		case 32:
			return Int[int32]{int32(n)}
		case 64:
			return Int[int64]{int64(n)}
		default:
			panic("cannot represent constant")
		}
	} else {
		n, exact := constant.Uint64Val(constant.ToInt(k.Value))
		if !exact {
			panic("cannot represent constant")
		}
		width := int(std.Sizeof(typ)) * 8
		switch width {
		case 8:
			return Uint[uint8]{uint8(n)}
		case 16:
			return Uint[uint16]{uint16(n)}
		case 32:
			return Uint[uint32]{uint32(n)}
		case 64:
			return Uint[uint64]{uint64(n)}
		default:
			panic("cannot represent constant")
		}
	}
}
