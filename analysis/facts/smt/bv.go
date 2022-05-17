package smt

// XXX consider the interaction of path predicates and equality propagation

import (
	"fmt"
	"go/constant"
)

type Node interface {
	String() string
}

type Var struct {
	Name uint64
}

func (v Var) String() string {
	return fmt.Sprintf("$%d", v.Name)
}

type Const struct {
	Value constant.Value
}

func (c Const) String() string {
	return c.Value.ExactString()
}

type Verb int

var verbs = map[Verb]string{
	verbAnd:    "and",
	verbOr:     "or",
	verbXor:    "xor",
	verbEqual:  "=",
	verbNot:    "not",
	verbBvneg:  "bvneg",
	verbBvadd:  "bvadd",
	verbBvsub:  "bvsub",
	verbBvmul:  "bvmul",
	verbBvshl:  "bvshl",
	verbBvult:  "bvult",
	verbBvslt:  "bvslt",
	verbBvule:  "bvule",
	verbBvsle:  "bvsle",
	verbBvlshr: "bvlshr",
	verbBvashr: "bvashr",
	verbBvand:  "bvand",
	verbBvor:   "bvor",
	verbBvxor:  "bvxor",
	verbBvudiv: "bvudiv",
	verbBvsdiv: "bvsdiv",
	verbBvurem: "bvurem",
	verbBvsrem: "bvsrem",
	verbBvnot:  "bvnot",
}

func (v Verb) String() string {
	return verbs[v]
}

const (
	verbAnd = iota
	verbOr
	verbXor
	verbEqual
	verbNot
	verbBvneg
	verbBvadd
	verbBvsub
	verbBvmul
	verbBvshl
	verbBvult
	verbBvslt
	verbBvule
	verbBvsle
	verbBvlshr
	verbBvashr
	verbBvand
	verbBvor
	verbBvxor
	verbBvudiv
	verbBvsdiv
	verbBvurem
	verbBvsrem
	verbBvnot
)

type Sexp struct {
	Verb Verb
	In   [2]Node
}

func (s Sexp) String() string {
	if s.In[1] == nil {
		return fmt.Sprintf("(%s %s)", s.Verb, s.In[0])
	} else {
		return fmt.Sprintf("(%s %s %s)", s.Verb, s.In[0], s.In[1])
	}
}

type key [2]any

func And(nodes ...Node) Node {
	switch len(nodes) {
	case 0:
		return Const{constant.MakeBool(true)}
	case 1:
		return nodes[0]
	default:
		and := Op(verbAnd, nodes[0], nodes[1])
		for _, n := range nodes[2:] {
			and = Op(verbAnd, n, and)
		}
		return and
	}
}

func Or(nodes ...Node) Node {
	switch len(nodes) {
	case 0:
		return Const{constant.MakeBool(false)}
	case 1:
		return nodes[0]
	default:
		or := Op(verbOr, nodes[0], nodes[1])
		for _, n := range nodes[2:] {
			or = Op(verbOr, n, or)
		}
		return or
	}
}

func Xor(a, b Node) Node {
	return Op(verbXor, a, b)
}

func Equal(a, b Node) Node {
	return Op(verbEqual, a, b)
}

func Not(a Node) Node {
	return Op(verbNot, a, nil)
}

func Op(verb Verb, a, b Node) Node {
	return Sexp{
		Verb: verb,
		In:   [2]Node{a, b},
	}
}
