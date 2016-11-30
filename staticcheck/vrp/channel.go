package vrp

import (
	"fmt"
	"math/big"

	"honnef.co/go/ssa"
)

type ChannelInterval struct {
	Size IntInterval
}

func (c ChannelInterval) Union(other Range) Range {
	i, ok := other.(ChannelInterval)
	if !ok {
		i = ChannelInterval{EmptyIntInterval}
	}
	if c.Size.Empty() || !c.Size.IsKnown() {
		return i
	}
	if i.Size.Empty() || !i.Size.IsKnown() {
		return c
	}
	return ChannelInterval{
		Size: c.Size.Union(i.Size).(IntInterval),
	}
}

func (c ChannelInterval) String() string {
	return c.Size.String()
}

func (c ChannelInterval) IsKnown() bool {
	return c.Size.IsKnown()
}

type MakeChannelConstraint struct {
	aConstraint
	Buffer ssa.Value
}

func (c *MakeChannelConstraint) String() string {
	return fmt.Sprintf("%s = make(chan, %s)", c.Y().Name, c.Buffer.Name())
}

func (c *MakeChannelConstraint) Eval(g *Graph) Range {
	i, ok := g.Range(c.Buffer).(IntInterval)
	if !ok {
		return ChannelInterval{NewIntInterval(NewZ(&big.Int{}), PInfinity)}
	}
	if i.Lower.Sign() == -1 {
		i.Lower = NewZ(&big.Int{})
	}
	return ChannelInterval{i}
}

func (c *MakeChannelConstraint) Operands() []ssa.Value {
	return []ssa.Value{c.Buffer}
}

type ChannelChangeTypeConstraint struct {
	aConstraint
	X ssa.Value
}

func (c *ChannelChangeTypeConstraint) String() string {
	return fmt.Sprintf("%s = changetype(%s)", c.Y().Name, c.X.Name())
}

func (c *ChannelChangeTypeConstraint) Eval(g *Graph) Range {
	return g.Range(c.X)
}

func (c *ChannelChangeTypeConstraint) Operands() []ssa.Value {
	return []ssa.Value{c.X}
}
