package circuit

import "fmt"

type Node struct {
	IsVar    bool // is this a Var or an And
	Inverted bool
	Var      uint64
	And      *and
}

func (n Node) String() string {
	if n.IsVar {
		if n.Inverted {
			return fmt.Sprintf("~%d", n.Var)
		} else {
			return fmt.Sprintf("%d", n.Var)
		}
	} else {
		if n.Inverted {
			return fmt.Sprintf("~(%s & %s)", n.And.X, n.And.Y)
		} else {
			return fmt.Sprintf("(%s & %s)", n.And.X, n.And.Y)
		}
	}
}

type and struct {
	X Node
	Y Node
}

func Var(x uint64) Node {
	return Node{
		IsVar: true,
		Var:   x,
	}
}

func And(ns ...Node) Node {
	if len(ns) == 0 {
		panic("XXX")
	}
	if len(ns) == 1 {
		return ns[0]
	}

	out := Node{
		And: &and{ns[0], ns[1]},
	}
	for _, n := range ns[2:] {
		out = Node{
			And: &and{out, n},
		}
	}
	return out
}

func Not(x Node) Node {
	x.Inverted = !x.Inverted
	return x
}

func Or(ns ...Node) Node {
	if len(ns) == 0 {
		panic("XXX")
	}
	if len(ns) == 1 {
		return ns[0]
	}
	out := Nand(
		Not(ns[0]),
		Not(ns[1]))
	for _, n := range ns[2:] {
		out = Nand(
			Not(out),
			Not(n))
	}
	return out
}

func Nand(x, y Node) Node {
	return Not(And(x, y))
}

func Equal(x, y Node) Node {
	return Or(
		And(x, y),
		And(Not(x), Not(y)))
}
