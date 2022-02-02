package vrp

import (
	"fmt"
	"go/constant"
	"go/types"

	"honnef.co/go/tools/go/ir"
)

type Int struct {
	n     int64
	inf   int8 // -1 = -∞, 1 = ∞
	width int8 // < 0 = unsigned, > 0 = signed
}

var Inf = &Int{inf: 1}
var NegInf = &Int{inf: -1}

func (n *Int) Add(o *Int) (*Int, bool) {
	if n.inf == 0 && o.inf != 0 {
		return o, false
	}
	if n.inf != 0 && o.inf == 0 {
		return n, false
	}
	if n.inf != 0 && n.inf == o.inf {
		return n, false
	}
	if n.inf != 0 && n.inf != o.inf {
		panic("-∞ + ∞ is not defined")
	}

	if n.width < 0 {
		r := uint64(n.n) + uint64(o.n)
		var max uint64 = 1<<(-n.width) - 1
		of := r < uint64(n.n) || r > max
		return &Int{n: int64(r), width: n.width}, of
	} else {
		var min int64 = -1 << (n.width - 1)
		var max int64 = 1<<(n.width-1) - 1
		r := n.n + o.n
		of := (r > n.n) != (o.n > 0) || r < min || r > max
		return &Int{n: r, width: n.width}, of
	}
}

func (n *Int) Sub(o *Int) (*Int, bool) {
	if n.inf != 0 && o.inf == 0 {
		return n, false
	}
	if n.inf == 1 && o.inf == 1 {
		panic("∞ - ∞ not defined")
	}
	if n.inf == 1 && o.inf == -1 {
		return Inf, false
	}
	if n.inf == -1 && o.inf == 1 {
		return NegInf, false
	}
	if n.inf == -1 && o.inf == -1 {
		panic("-∞ + ∞ not defined")
	}
	if n.inf == 0 && o.inf == 1 {
		return NegInf, false
	}
	if n.inf == 0 && o.inf == -1 {
		return Inf, false
	}

	if n.width < 0 {
		var max uint64 = 1<<(-n.width) - 1
		r := uint64(n.n) - uint64(o.n)
		of := r > uint64(n.n) || r > max
		return &Int{n: int64(r), width: n.width}, of
	} else {
		var min int64 = -1 << (n.width - 1)
		var max int64 = 1<<(n.width-1) - 1
		r := n.n - o.n
		of := (r < n.n) != (o.n > 0) || r < min || r > max
		return &Int{n: r, width: n.width}, of
	}
}

func (n *Int) Mul(o *Int) (*Int, bool) {
	if inf := n.inf * o.inf; inf != 0 {
		return &Int{inf: inf}, false
	}

	if n.n == 0 {
		return n, false
	}
	if o.n == 0 {
		return o, false
	}

	if n.width < 0 {
		var max uint64 = 1<<(-n.width) - 1
		r := uint64(n.n) * uint64(o.n)
		of := r/uint64(o.n) != uint64(n.n) || r > max
		return &Int{n: int64(r), width: n.width}, of
	} else {
		var min int64 = -1 << (n.width - 1)
		var max int64 = 1<<(n.width-1) - 1
		r := n.n * o.n
		of := (r < 0) != ((n.n < 0) != (o.n < 0)) || r/o.n != n.n || r < min || r > max
		return &Int{n: r, width: n.width}, of
	}
}

func (n *Int) Cmp(o *Int) int {
	if n.inf != 0 && n.inf == o.inf {
		return 0
	}
	if n.inf == -1 {
		return -1
	}
	if n.inf == 1 {
		return 1
	}
	if o.inf == -1 {
		return 1
	}
	if o.inf == 1 {
		return -1
	}

	if n.width < 0 {
		if uint64(n.n) > uint64(o.n) {
			return 1
		} else if n.n == o.n {
			return 0
		} else {
			return -1
		}
	} else {
		if n.n > o.n {
			return 1
		} else if n.n == o.n {
			return 0
		} else {
			return -1
		}
	}
}

func (n *Int) Dec() (*Int, bool) { return n.Sub(&Int{n: 1, width: n.width}) }
func (n *Int) Inc() (*Int, bool) { return n.Add(&Int{n: 1, width: n.width}) }
func (n *Int) Negative() bool    { return n.width > 0 && n.n < 0 }
func (n *Int) Infinite() int     { return int(n.inf) }
func (n *Int) String() string {
	switch n.inf {
	case -1:
		return "-∞"
	case 1:
		return "∞"
	case 0:
		return fmt.Sprintf("%d", n.n)
	default:
		panic("unreachable")
	}
}

func ConstToNumeric(k *ir.Const) *Int {
	typ := k.Type().Underlying().(*types.Basic)
	// XXX don't assume 64 bit
	std := types.StdSizes{WordSize: 8, MaxAlign: 1}
	if (typ.Info() & types.IsUnsigned) == 0 {
		n, exact := constant.Int64Val(constant.ToInt(k.Value))
		if !exact {
			panic("cannot represent constant")
		}
		width := int8(std.Sizeof(typ)) * 8
		return &Int{n: n, width: width}
	} else {
		n, exact := constant.Uint64Val(constant.ToInt(k.Value))
		if !exact {
			panic("cannot represent constant")
		}
		width := int8(std.Sizeof(typ)) * 8
		return &Int{n: int64(n), width: -width}
	}
}

func intWidth(typ types.Type) int8 {
	// XXX don't assume 64 bit
	std := types.StdSizes{WordSize: 8, MaxAlign: 1}
	return int8(std.Sizeof(typ)) * 8
}

func NewInt(v uint64, typ types.Type) *Int {
	width := intWidth(typ)
	if (typ.Underlying().(*types.Basic).Info() & types.IsUnsigned) != 0 {
		width = -width
	}
	return &Int{n: int64(v), width: width}
}
