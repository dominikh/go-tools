package smt

// XXX consider the interaction of path predicates and equality propagation

import (
	"fmt"
	"go/constant"
	"go/types"
	"strings"
)

func makeValue(typ Type) value                   { return value{typ} }
func MakeVar(typ Type, name uint64) Var          { return Var{makeValue(typ), name} }
func MakeConst(typ Type, v constant.Value) Const { return Const{makeValue(typ), v} }

func fromGoType(typ types.Type) Type {
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok {
		// XXX
		panic(fmt.Sprintf("unsupported type %T", typ))
	}

	switch basic.Kind() {
	// predeclared types
	case types.Bool:
		return Bool{}
	case types.Int, types.Uint, types.Uintptr:
		// XXX don't assume 64 bits
		return BitVector{64}
	case types.Int8, types.Uint8:
		return BitVector{8}
	case types.Int16, types.Uint16:
		return BitVector{16}
	case types.Int32, types.Uint32:
		return BitVector{32}
	case types.Int64, types.Uint64:
		return BitVector{64}
	default:
		// XXX
		panic(fmt.Sprintf("unsupported type %T", typ))
	}
}

type Type interface {
	Equal(o Type) bool
}

type Bool struct{}

func (b Bool) Equal(o Type) bool {
	_, ok := o.(Bool)
	return ok
}

type BitVector struct {
	// Size in bits
	Size int
}

func (bv BitVector) Equal(o Type) bool {
	obv, ok := o.(BitVector)
	return ok && obv.Size == bv.Size
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
	return fmt.Sprintf("v%d", v.Name)
}

type Sexp struct {
	value
	Verb Verb
	In   []Value
}

func (s *Sexp) String() string {
	args := make([]string, len(s.In))
	for i, in := range s.In {
		if in == nil {
			panic(fmt.Sprintf("nil input to sexp %#v", s))
		}
		args[i] = in.String()
	}
	return fmt.Sprintf("(%s %s)", s.Verb, strings.Join(args, " "))
}

type Const struct {
	value
	Value constant.Value
}

func (c Const) String() string {
	return c.Value.String()
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
	verbNone Verb = iota
	verbAnd
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
	return Op(Bool{}, verbAnd, nodes...)
}

func Or(nodes ...Value) *Sexp {
	return Op(Bool{}, verbOr, nodes...)
}

func Xor(a, b Value) *Sexp {
	return Op(Bool{}, verbXor, a, b)
}

func Equal(a, b Value) *Sexp {
	return Op(Bool{}, verbEqual, a, b)
}

func Not(a Value) *Sexp {
	return Op(Bool{}, verbNot, a)
}

func Op(typ Type, verb Verb, in ...Value) *Sexp {
	return &Sexp{
		value: makeValue(typ),
		Verb:  verb,
		In:    in,
	}
}

func Identity(v Value) Sexp {
	return Sexp{
		value: makeValue(v.Type()),
		Verb:  verbIdentity,
		In:    []Value{v},
	}
}
