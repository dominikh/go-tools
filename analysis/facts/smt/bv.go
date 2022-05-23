package smt

// XXX consider the interaction of path predicates and equality propagation

import (
	"fmt"
	"go/constant"
	"strings"
)

type Type interface{}

type Bool struct{}
type BitVector struct {
	Size int
}

type Value interface {
	Type() Type
	String() string
}

type value struct {
	typ Type
}

func (v value) Type() Type {
	return v.typ
}

type Var struct {
	value
	Name uint64
}

func (v Var) String() string {
	return fmt.Sprintf("v%d", v)
}

type Sexp struct {
	value
	Verb Verb
	In   []Value
}

func (s *Sexp) String() string {
	args := make([]string, len(s.In))
	for i, in := range s.In {
		args[i] = in.String()
	}
	return fmt.Sprintf("(%s %s)", s.Verb, strings.Join(args, " "))
}

type Const struct {
	value
	Const constant.Value
}

type Verb int

var verbs = map[Verb]string{
	verbAnd:      "and",
	verbOr:       "or",
	verbXor:      "xor",
	verbEqual:    "=",
	verbNot:      "not",
	verbIdentity: "identity",
	verbBvneg:    "bvneg",
	verbBvadd:    "bvadd",
	verbBvsub:    "bvsub",
	verbBvmul:    "bvmul",
	verbBvshl:    "bvshl",
	verbBvult:    "bvult",
	verbBvslt:    "bvslt",
	verbBvule:    "bvule",
	verbBvsle:    "bvsle",
	verbBvlshr:   "bvlshr",
	verbBvashr:   "bvashr",
	verbBvand:    "bvand",
	verbBvor:     "bvor",
	verbBvxor:    "bvxor",
	verbBvudiv:   "bvudiv",
	verbBvsdiv:   "bvsdiv",
	verbBvurem:   "bvurem",
	verbBvsrem:   "bvsrem",
	verbBvnot:    "bvnot",
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
	verbIdentity
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

func And(nodes ...Value) *Sexp {
	return Op(verbAnd, nodes...)
}

func Or(nodes ...Value) *Sexp {
	return Op(verbOr, nodes...)
}

func Xor(a, b Value) *Sexp {
	return Op(verbXor, a, b)
}

func Equal(a, b Value) *Sexp {
	return Op(verbEqual, a, b)
}

func Not(a Value) *Sexp {
	return Op(verbNot, a, nil)
}

func Op(verb Verb, in ...Value) *Sexp {
	return &Sexp{
		Verb: verb,
		In:   in,
	}
}

func Identity(v Value) Sexp {
	return Sexp{Verb: verbIdentity, v}
}
