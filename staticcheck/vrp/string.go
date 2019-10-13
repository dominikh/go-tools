package vrp

import (
	"fmt"
	"go/token"
	"go/types"

	"honnef.co/go/tools/ir"
)

type StringInterval struct {
	Length IntInterval
}

func (s StringInterval) Union(other Range) Range {
	i, ok := other.(StringInterval)
	if !ok {
		i = StringInterval{EmptyIntInterval}
	}
	if s.Length.Empty() || !s.Length.IsKnown() {
		return i
	}
	if i.Length.Empty() || !i.Length.IsKnown() {
		return s
	}
	return StringInterval{
		Length: s.Length.Union(i.Length).(IntInterval),
	}
}

func (s StringInterval) String() string {
	return s.Length.String()
}

func (s StringInterval) IsKnown() bool {
	return s.Length.IsKnown()
}

type StringSliceConstraint struct {
	aConstraint
	X     ir.Value
	Lower ir.Value
	Upper ir.Value
}

type StringIntersectionConstraint struct {
	aConstraint
	ranges   Ranges
	A        ir.Value
	B        ir.Value
	Op       token.Token
	I        IntInterval
	resolved bool
}

type StringConcatConstraint struct {
	aConstraint
	A ir.Value
	B ir.Value
}

type StringLengthConstraint struct {
	aConstraint
	X ir.Value
}

type StringIntervalConstraint struct {
	aConstraint
	I IntInterval
}

func NewStringSliceConstraint(x, lower, upper, y ir.Value) Constraint {
	return &StringSliceConstraint{NewConstraint(y), x, lower, upper}
}
func NewStringIntersectionConstraint(a, b ir.Value, op token.Token, ranges Ranges, y ir.Value) Constraint {
	return &StringIntersectionConstraint{
		aConstraint: NewConstraint(y),
		ranges:      ranges,
		A:           a,
		B:           b,
		Op:          op,
	}
}
func NewStringConcatConstraint(a, b, y ir.Value) Constraint {
	return &StringConcatConstraint{NewConstraint(y), a, b}
}
func NewStringLengthConstraint(x ir.Value, y ir.Value) Constraint {
	return &StringLengthConstraint{NewConstraint(y), x}
}
func NewStringIntervalConstraint(i IntInterval, y ir.Value) Constraint {
	return &StringIntervalConstraint{NewConstraint(y), i}
}

func (c *StringSliceConstraint) Operands() []ir.Value {
	vs := []ir.Value{c.X}
	if c.Lower != nil {
		vs = append(vs, c.Lower)
	}
	if c.Upper != nil {
		vs = append(vs, c.Upper)
	}
	return vs
}
func (c *StringIntersectionConstraint) Operands() []ir.Value { return []ir.Value{c.A} }
func (c StringConcatConstraint) Operands() []ir.Value        { return []ir.Value{c.A, c.B} }
func (c *StringLengthConstraint) Operands() []ir.Value       { return []ir.Value{c.X} }
func (s *StringIntervalConstraint) Operands() []ir.Value     { return nil }

func (c *StringSliceConstraint) String() string {
	var lname, uname string
	if c.Lower != nil {
		lname = c.Lower.Name()
	}
	if c.Upper != nil {
		uname = c.Upper.Name()
	}
	return fmt.Sprintf("%s[%s:%s]", c.X.Name(), lname, uname)
}
func (c *StringIntersectionConstraint) String() string {
	panic("unreachable")
}
func (c StringConcatConstraint) String() string {
	return fmt.Sprintf("%s = %s + %s", c.Y().Name(), c.A.Name(), c.B.Name())
}
func (c *StringLengthConstraint) String() string {
	return fmt.Sprintf("%s = len(%s)", c.Y().Name(), c.X.Name())
}
func (c *StringIntervalConstraint) String() string { return fmt.Sprintf("%s = %s", c.Y().Name(), c.I) }

func (c *StringSliceConstraint) Eval(g *Graph) Range {
	lr := NewIntInterval(NewZ(0), NewZ(0))
	if c.Lower != nil {
		lr = g.Range(c.Lower).(IntInterval)
	}
	ur := g.Range(c.X).(StringInterval).Length
	if c.Upper != nil {
		ur = g.Range(c.Upper).(IntInterval)
	}
	if !lr.IsKnown() || !ur.IsKnown() {
		return StringInterval{}
	}

	ls := []Z{
		ur.Lower.Sub(lr.Lower),
		ur.Upper.Sub(lr.Lower),
		ur.Lower.Sub(lr.Upper),
		ur.Upper.Sub(lr.Upper),
	}
	// TODO(dh): if we don't truncate lengths to 0 we might be able to
	// easily detect slices with high < low. we'd need to treat -∞
	// specially, though.
	for i, l := range ls {
		if l.Sign() == -1 {
			ls[i] = NewZ(0)
		}
	}

	return StringInterval{
		Length: NewIntInterval(MinZ(ls...), MaxZ(ls...)),
	}
}
func (c *StringIntersectionConstraint) Eval(g *Graph) Range {
	var l IntInterval
	switch r := g.Range(c.A).(type) {
	case StringInterval:
		l = r.Length
	case IntInterval:
		l = r
	}

	if !l.IsKnown() {
		return StringInterval{c.I}
	}
	return StringInterval{
		Length: l.Intersection(c.I),
	}
}
func (c StringConcatConstraint) Eval(g *Graph) Range {
	i1, i2 := g.Range(c.A).(StringInterval), g.Range(c.B).(StringInterval)
	if !i1.Length.IsKnown() || !i2.Length.IsKnown() {
		return StringInterval{}
	}
	return StringInterval{
		Length: i1.Length.Add(i2.Length),
	}
}
func (c *StringLengthConstraint) Eval(g *Graph) Range {
	i := g.Range(c.X).(StringInterval).Length
	if !i.IsKnown() {
		return NewIntInterval(NewZ(0), PInfinity)
	}
	return i
}
func (c *StringIntervalConstraint) Eval(*Graph) Range { return StringInterval{c.I} }

func (c *StringIntersectionConstraint) Futures() []ir.Value {
	return []ir.Value{c.B}
}

func (c *StringIntersectionConstraint) Resolve() {
	if (c.A.Type().Underlying().(*types.Basic).Info() & types.IsString) != 0 {
		// comparing two strings
		r, ok := c.ranges[c.B].(StringInterval)
		if !ok {
			c.I = NewIntInterval(NewZ(0), PInfinity)
			return
		}
		switch c.Op {
		case token.EQL:
			c.I = r.Length
		case token.GTR, token.GEQ:
			c.I = NewIntInterval(r.Length.Lower, PInfinity)
		case token.LSS, token.LEQ:
			c.I = NewIntInterval(NewZ(0), r.Length.Upper)
		case token.NEQ:
		default:
			panic("unsupported op " + c.Op.String())
		}
	} else {
		r, ok := c.ranges[c.B].(IntInterval)
		if !ok {
			c.I = NewIntInterval(NewZ(0), PInfinity)
			return
		}
		// comparing two lengths
		switch c.Op {
		case token.EQL:
			c.I = r
		case token.GTR:
			c.I = NewIntInterval(r.Lower.Add(NewZ(1)), PInfinity)
		case token.GEQ:
			c.I = NewIntInterval(r.Lower, PInfinity)
		case token.LSS:
			c.I = NewIntInterval(NInfinity, r.Upper.Sub(NewZ(1)))
		case token.LEQ:
			c.I = NewIntInterval(NInfinity, r.Upper)
		case token.NEQ:
		default:
			panic("unsupported op " + c.Op.String())
		}
	}
}

func (c *StringIntersectionConstraint) IsKnown() bool {
	return c.I.IsKnown()
}

func (c *StringIntersectionConstraint) MarkUnresolved() {
	c.resolved = false
}

func (c *StringIntersectionConstraint) MarkResolved() {
	c.resolved = true
}

func (c *StringIntersectionConstraint) IsResolved() bool {
	return c.resolved
}
