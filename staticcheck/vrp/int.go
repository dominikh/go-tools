package vrp

import (
	"fmt"
	"go/token"
	"go/types"
	"math/big"

	"honnef.co/go/ssa"
)

type Zs []Z

func (zs Zs) Len() int {
	return len(zs)
}

func (zs Zs) Less(i int, j int) bool {
	return zs[i].Cmp(zs[j]) == -1
}

func (zs Zs) Swap(i int, j int) {
	zs[i], zs[j] = zs[j], zs[i]
}

type Z struct {
	infinity int8
	integer  *big.Int
}

func NewZ(n int64) Z {
	return NewBigZ(big.NewInt(n))
}

func NewBigZ(n *big.Int) Z {
	return Z{integer: n}
}

func (z1 Z) Infinite() bool {
	return z1.infinity != 0
}

func (z1 Z) Add(z2 Z) Z {
	if z2.Sign() == -1 {
		return z1.Sub(z2.Negate())
	}
	if z1 == NInfinity {
		return NInfinity
	}
	if z1 == PInfinity {
		return PInfinity
	}
	if z2 == PInfinity {
		return PInfinity
	}

	if !z1.Infinite() && !z2.Infinite() {
		n := &big.Int{}
		n.Add(z1.integer, z2.integer)
		return NewBigZ(n)
	}

	panic(fmt.Sprintf("%s + %s is not defined", z1, z2))
}

func (z1 Z) Sub(z2 Z) Z {
	if z2.Sign() == -1 {
		return z1.Add(z2.Negate())
	}
	if !z1.Infinite() && !z2.Infinite() {
		n := &big.Int{}
		n.Sub(z1.integer, z2.integer)
		return NewBigZ(n)
	}

	if z1 != PInfinity && z2 == PInfinity {
		return NInfinity
	}
	if z1.Infinite() && !z2.Infinite() {
		return Z{infinity: z1.infinity}
	}
	if z1 == PInfinity && z2 == PInfinity {
		return PInfinity
	}
	panic(fmt.Sprintf("%s - %s is not defined", z1, z2))
}

func (z1 Z) Mul(z2 Z) Z {
	if (z1.integer != nil && z1.integer.Sign() == 0) ||
		(z2.integer != nil && z2.integer.Sign() == 0) {
		return NewBigZ(&big.Int{})
	}

	if z1.infinity != 0 || z2.infinity != 0 {
		return Z{infinity: int8(z1.Sign() * z2.Sign())}
	}

	n := &big.Int{}
	n.Mul(z1.integer, z2.integer)
	return NewBigZ(n)
}

func (z1 Z) Negate() Z {
	if z1.infinity == 1 {
		return NInfinity
	}
	if z1.infinity == -1 {
		return PInfinity
	}
	n := &big.Int{}
	n.Neg(z1.integer)
	return NewBigZ(n)
}

func (z1 Z) Sign() int {
	if z1.infinity != 0 {
		return int(z1.infinity)
	}
	return z1.integer.Sign()
}

func (z1 Z) String() string {
	if z1 == NInfinity {
		return "-∞"
	}
	if z1 == PInfinity {
		return "∞"
	}
	return fmt.Sprintf("%d", z1.integer)
}

func (z1 Z) Cmp(z2 Z) int {
	if z1.infinity == z2.infinity && z1.infinity != 0 {
		return 0
	}
	if z1 == PInfinity {
		return 1
	}
	if z1 == NInfinity {
		return -1
	}
	if z2 == NInfinity {
		return 1
	}
	if z2 == PInfinity {
		return -1
	}
	return z1.integer.Cmp(z2.integer)
}

func MaxZ(zs ...Z) Z {
	if len(zs) == 0 {
		panic("Max called with no arguments")
	}
	if len(zs) == 1 {
		return zs[0]
	}
	ret := zs[0]
	for _, z := range zs[1:] {
		if z.Cmp(ret) == 1 {
			ret = z
		}
	}
	return ret
}

func MinZ(zs ...Z) Z {
	if len(zs) == 0 {
		panic("Min called with no arguments")
	}
	if len(zs) == 1 {
		return zs[0]
	}
	ret := zs[0]
	for _, z := range zs[1:] {
		if z.Cmp(ret) == -1 {
			ret = z
		}
	}
	return ret
}

var NInfinity = Z{infinity: -1}
var PInfinity = Z{infinity: 1}
var EmptyIntInterval = IntInterval{true, PInfinity, NInfinity}

func InfinityFor(v ssa.Value) IntInterval {
	if b, ok := v.Type().Underlying().(*types.Basic); ok {
		if (b.Info() & types.IsUnsigned) != 0 {
			return NewIntInterval(NewZ(0), PInfinity)
		}
	}
	return NewIntInterval(NInfinity, PInfinity)
}

type IntInterval struct {
	known bool
	Lower Z
	Upper Z
}

func NewIntInterval(l, u Z) IntInterval {
	if u.Cmp(l) == -1 {
		return EmptyIntInterval
	}
	return IntInterval{known: true, Lower: l, Upper: u}
}

func (i IntInterval) IsKnown() bool {
	return i.known
}

func (i IntInterval) Empty() bool {
	return i.Lower == PInfinity && i.Upper == NInfinity
}

func (i IntInterval) IsMaxRange() bool {
	return i.Lower == NInfinity && i.Upper == PInfinity
}

func (i1 IntInterval) Intersection(i2 IntInterval) IntInterval {
	if !i1.IsKnown() {
		return i2
	}
	if !i2.IsKnown() {
		return i1
	}
	if i1.Empty() || i2.Empty() {
		return EmptyIntInterval
	}
	i3 := NewIntInterval(MaxZ(i1.Lower, i2.Lower), MinZ(i1.Upper, i2.Upper))
	if i3.Lower.Cmp(i3.Upper) == 1 {
		return EmptyIntInterval
	}
	return i3
}

func (i1 IntInterval) Union(other Range) Range {
	i2, ok := other.(IntInterval)
	if !ok {
		i2 = EmptyIntInterval
	}
	if i1.Empty() || !i1.IsKnown() {
		return i2
	}
	if i2.Empty() || !i2.IsKnown() {
		return i1
	}
	return NewIntInterval(MinZ(i1.Lower, i2.Lower), MaxZ(i1.Upper, i2.Upper))
}

func (i1 IntInterval) Add(i2 IntInterval) IntInterval {
	if i1.Empty() || i2.Empty() {
		return EmptyIntInterval
	}
	l1, u1, l2, u2 := i1.Lower, i1.Upper, i2.Lower, i2.Upper
	return NewIntInterval(l1.Add(l2), u1.Add(u2))
}

func (i1 IntInterval) Sub(i2 IntInterval) IntInterval {
	if i1.Empty() || i2.Empty() {
		return EmptyIntInterval
	}
	l1, u1, l2, u2 := i1.Lower, i1.Upper, i2.Lower, i2.Upper
	return NewIntInterval(l1.Sub(u2), u1.Sub(l2))
}

func (i1 IntInterval) Mul(i2 IntInterval) IntInterval {
	if i1.Empty() || i2.Empty() {
		return EmptyIntInterval
	}
	x1, x2 := i1.Lower, i1.Upper
	y1, y2 := i2.Lower, i2.Upper
	return NewIntInterval(
		MinZ(x1.Mul(y1), x1.Mul(y2), x2.Mul(y1), x2.Mul(y2)),
		MaxZ(x1.Mul(y1), x1.Mul(y2), x2.Mul(y1), x2.Mul(y2)),
	)
}

func (i1 IntInterval) String() string {
	if !i1.IsKnown() {
		return "[⊥, ⊥]"
	}
	if i1.Empty() {
		return "{}"
	}
	return fmt.Sprintf("[%s, %s]", i1.Lower, i1.Upper)
}

type IntArithmeticConstraint struct {
	aConstraint
	A  ssa.Value
	B  ssa.Value
	Op token.Token
	Fn func(IntInterval, IntInterval) IntInterval
}

type IntAddConstraint struct{ *IntArithmeticConstraint }
type IntSubConstraint struct{ *IntArithmeticConstraint }
type IntMulConstraint struct{ *IntArithmeticConstraint }

type IntConversionConstraint struct {
	aConstraint
	X ssa.Value
}

type FutureIntIntersectionConstraint struct {
	aConstraint
	ranges      map[ssa.Value]Range
	X           ssa.Value
	lower       ssa.Value
	lowerOffset Z
	upper       ssa.Value
	upperOffset Z
	I           IntInterval
	resolved    bool
}

type IntIntersectionConstraint struct {
	aConstraint
	X ssa.Value
	I IntInterval
}

type IntIntervalConstraint struct {
	aConstraint
	I IntInterval
}

func NewIntArithmeticConstraint(a, b, y ssa.Value, op token.Token, fn func(IntInterval, IntInterval) IntInterval) *IntArithmeticConstraint {
	return &IntArithmeticConstraint{NewConstraint(y), a, b, op, fn}
}
func NewIntAddConstraint(a, b, y ssa.Value) Constraint {
	return &IntAddConstraint{NewIntArithmeticConstraint(a, b, y, token.ADD, IntInterval.Add)}
}
func NewIntSubConstraint(a, b, y ssa.Value) Constraint {
	return &IntSubConstraint{NewIntArithmeticConstraint(a, b, y, token.SUB, IntInterval.Sub)}
}
func NewIntMulConstraint(a, b, y ssa.Value) Constraint {
	return &IntMulConstraint{NewIntArithmeticConstraint(a, b, y, token.MUL, IntInterval.Mul)}
}
func NewIntConversionConstraint(x, y ssa.Value) Constraint {
	return &IntConversionConstraint{NewConstraint(y), x}
}
func NewIntIntersectionConstraint(x ssa.Value, i IntInterval, y ssa.Value) Constraint {
	return &IntIntersectionConstraint{NewConstraint(y), x, i}
}
func NewIntIntervalConstraint(i IntInterval, y ssa.Value) Constraint {
	return &IntIntervalConstraint{NewConstraint(y), i}
}

func (c *IntArithmeticConstraint) Operands() []ssa.Value         { return []ssa.Value{c.A, c.B} }
func (c *IntConversionConstraint) Operands() []ssa.Value         { return []ssa.Value{c.X} }
func (c *FutureIntIntersectionConstraint) Operands() []ssa.Value { return []ssa.Value{c.X} }
func (c *IntIntersectionConstraint) Operands() []ssa.Value       { return []ssa.Value{c.X} }
func (s *IntIntervalConstraint) Operands() []ssa.Value           { return nil }

func (c *IntArithmeticConstraint) String() string {
	return fmt.Sprintf("%s = %s %s %s", c.Y().Name(), c.A.Name(), c.Op, c.B.Name())
}
func (c *IntConversionConstraint) String() string {
	return fmt.Sprintf("%s = %s(%s)", c.Y().Name(), c.Y().Type(), c.X.Name())
}
func (c *FutureIntIntersectionConstraint) String() string {
	var lname, uname string
	if c.lower != nil {
		lname = c.lower.Name()
	}
	if c.upper != nil {
		uname = c.upper.Name()
	}
	return fmt.Sprintf("%s = %s.%t ⊓ [%s+%s, %s+%s] %s",
		c.Y().Name(), c.X.Name(), c.Y().(*ssa.Sigma).Branch, lname, c.lowerOffset, uname, c.upperOffset, c.I)
}
func (c *IntIntersectionConstraint) String() string {
	return fmt.Sprintf("%s = %s.%t ⊓ %s", c.Y().Name(), c.X.Name(), c.Y().(*ssa.Sigma).Branch, c.I)
}
func (c *IntIntervalConstraint) String() string { return fmt.Sprintf("%s = %s", c.Y().Name(), c.I) }

func (c *IntArithmeticConstraint) Eval(g *Graph) Range {
	i1, i2 := g.Range(c.A).(IntInterval), g.Range(c.B).(IntInterval)
	if !i1.IsKnown() || !i2.IsKnown() {
		return IntInterval{}
	}
	return c.Fn(i1, i2)
}
func (c *IntConversionConstraint) Eval(g *Graph) Range {
	s := &types.StdSizes{
		// XXX is it okay to assume the largest word size, or do we
		// need to be platform specific?
		WordSize: 8,
		MaxAlign: 1,
	}
	fromI := g.Range(c.X).(IntInterval)
	toI := g.Range(c.Y()).(IntInterval)
	fromT := c.X.Type().Underlying().(*types.Basic)
	toT := c.Y().Type().Underlying().(*types.Basic)
	fromB := s.Sizeof(c.X.Type())
	toB := s.Sizeof(c.Y().Type())

	if !fromI.IsKnown() {
		return toI
	}
	if !toI.IsKnown() {
		return fromI
	}

	// uint<N> -> sint/uint<M>, M > N: [max(0, l1), min(2**N-1, u2)]
	if (fromT.Info()&types.IsUnsigned != 0) &&
		toB > fromB {

		n := big.NewInt(1)
		n.Lsh(n, uint(fromB*8))
		n.Sub(n, big.NewInt(1))
		return NewIntInterval(
			MaxZ(NewZ(0), fromI.Lower),
			MinZ(NewBigZ(n), toI.Upper),
		)
	}

	// sint<N> -> sint<M>, M > N; [max(-∞, l1), min(2**N-1, u2)]
	if (fromT.Info()&types.IsUnsigned == 0) &&
		(toT.Info()&types.IsUnsigned == 0) &&
		toB > fromB {

		n := big.NewInt(1)
		n.Lsh(n, uint(fromB*8))
		n.Sub(n, big.NewInt(1))
		return NewIntInterval(
			MaxZ(NInfinity, fromI.Lower),
			MinZ(NewBigZ(n), toI.Upper),
		)
	}

	return fromI
}
func (c *FutureIntIntersectionConstraint) Eval(g *Graph) Range {
	xi := g.Range(c.X).(IntInterval)
	return xi.Intersection(c.I)
}
func (c *IntIntersectionConstraint) Eval(g *Graph) Range {
	xi := g.Range(c.X).(IntInterval)
	if !xi.IsKnown() {
		return c.I
	}
	return xi.Intersection(c.I)
}
func (c *IntIntervalConstraint) Eval(*Graph) Range { return c.I }

func (c *FutureIntIntersectionConstraint) Futures() []ssa.Value {
	var s []ssa.Value
	if c.lower != nil {
		s = append(s, c.lower)
	}
	if c.upper != nil {
		s = append(s, c.upper)
	}
	return s
}

func (c *FutureIntIntersectionConstraint) Resolve() {
	l := NInfinity
	u := PInfinity
	if c.lower != nil {
		li, ok := c.ranges[c.lower]
		if !ok {
			li = InfinityFor(c.lower)
		}
		l = li.(IntInterval).Lower
		l = l.Add(c.lowerOffset)
	}
	if c.upper != nil {
		ui, ok := c.ranges[c.upper]
		if !ok {
			ui = InfinityFor(c.upper)
		}
		u = ui.(IntInterval).Upper
		u = u.Add(c.upperOffset)
	}
	c.I = NewIntInterval(l, u)
}
