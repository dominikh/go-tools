package vrp

import (
	"fmt"

	"honnef.co/go/ssa"
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
	X     ssa.Value
	Lower ssa.Value
	Upper ssa.Value
}

type StringIntersectionConstraint struct {
	aConstraint
	X ssa.Value
	I IntInterval
}

type StringConcatConstraint struct {
	aConstraint
	A ssa.Value
	B ssa.Value
}

type StringLengthConstraint struct {
	aConstraint
	X ssa.Value
}

type StringIntervalConstraint struct {
	aConstraint
	I IntInterval
}

func NewStringSliceConstraint(x, lower, upper, y ssa.Value) Constraint {
	return &StringSliceConstraint{NewConstraint(y), x, lower, upper}
}
func NewStringIntersectionConstraint(x ssa.Value, i IntInterval, y ssa.Value) Constraint {
	return &StringIntersectionConstraint{NewConstraint(y), x, i}
}
func NewStringConcatConstraint(a, b, y ssa.Value) Constraint {
	return &StringConcatConstraint{NewConstraint(y), a, b}
}
func NewStringLengthConstraint(x ssa.Value, y ssa.Value) Constraint {
	return &StringLengthConstraint{NewConstraint(y), x}
}
func NewStringIntervalConstraint(i IntInterval, y ssa.Value) Constraint {
	return &StringIntervalConstraint{NewConstraint(y), i}
}

func (c *StringSliceConstraint) Operands() []ssa.Value {
	vs := []ssa.Value{c.X}
	if c.Lower != nil {
		vs = append(vs, c.Lower)
	}
	if c.Upper != nil {
		vs = append(vs, c.Upper)
	}
	return vs
}
func (c *StringIntersectionConstraint) Operands() []ssa.Value { return []ssa.Value{c.X} }
func (c StringConcatConstraint) Operands() []ssa.Value        { return []ssa.Value{c.A, c.B} }
func (c *StringLengthConstraint) Operands() []ssa.Value       { return []ssa.Value{c.X} }
func (s *StringIntervalConstraint) Operands() []ssa.Value     { return nil }

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
	return fmt.Sprintf("%s = %s.%t ⊓ %s", c.Y().Name(), c.X.Name(), c.Y().(*ssa.Sigma).Branch, c.I)
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
	xi := g.Range(c.X).(StringInterval)
	if !xi.IsKnown() {
		return c.I
	}
	return StringInterval{
		Length: xi.Length.Intersection(c.I),
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
