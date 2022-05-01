package smt

// XXX consider the interaction of path predicates and equality propagation

import (
	"fmt"
	"go/constant"
	"go/token"
	"log"
	"strings"

	"honnef.co/go/tools/go/ir"
)

type Node interface {
	String() string
	Equal(Node) bool
}

type Const struct {
	Value constant.Value
}

func (c Const) Equal(o Node) bool {
	return c == o
}

func (c Const) String() string {
	// XXX emit valid sexp
	return c.Value.ExactString()
}

type Var struct {
	Value ir.Value
}

func (v Var) Equal(o Node) bool {
	return v == o
}

func (v Var) String() string {
	return v.Value.Name()
}

type Sexp struct {
	// OPT don't use string for Verb
	Verb string
	In   []Node
}

func (s Sexp) Equal(o Node) bool {
	so, ok := o.(Sexp)
	if !ok {
		return false
	}
	if s.Verb != so.Verb {
		return false
	}
	if len(s.In) != len(so.In) {
		return false
	}
	for i := range s.In {
		if !s.In[i].Equal(so.In[i]) {
			return false
		}
	}
	return true
}

func (sexp Sexp) String() string {
	args := make([]string, len(sexp.In))
	for i, arg := range sexp.In {
		args[i] = arg.String()
	}
	return fmt.Sprintf("(%s %s)", sexp.Verb, strings.Join(args, " "))
}

type key [2]any

type builder struct {
	vars       map[ir.Value]Node
	predicates map[ir.Value]Node
}

// TODO number nodes for a canonical ordering

func (b *builder) value(v ir.Value, n Node) {
	b.vars[v] = n
}

func (b *builder) predicate(v ir.Value, p Node) {
	b.predicates[v] = p
}

func And(nodes ...Node) Sexp {
	return Op("and", nodes...)
}

func Or(nodes ...Node) Sexp {
	return Op("or", nodes...)
}

func Equal(a, b Node) Sexp {
	return Op("=", a, b)
}

func Not(a Node) Sexp {
	return Op("not", a)
}

func Op(verb string, nodes ...Node) Sexp {
	return Sexp{
		Verb: verb,
		In:   nodes,
	}
}

// Our formulas are generally small enough that we don't care about on-the-fly simplifications. Instead use a simple
// fixpoint approach.

func (b *builder) simplify(n Node) Node {
	for {
		new, changed := b.simplify0(n)
		n = new
		if !changed {
			return n
		}
	}
}

func substitute(n Node, subst map[Var]Node, sameLevel bool) (Node, bool) {
	switch n := n.(type) {
	case Var:
		if s, ok := subst[n]; ok {
			return s, true
		} else {
			return n, false
		}
	case Sexp:
		changed := false
		new := make([]Node, len(n.In))
		for i, in := range n.In {
			var ok bool
			new[i], ok = substitute(in, subst, false)
			if i == 0 && n.Verb == "=" && new[i].Equal(n.In[1]) && sameLevel {
				new[i] = in
				ok = false
			}
			if ok {
				changed = true
			}
		}
		return Op(n.Verb, new...), changed
	default:
		return n, false
	}
}

func tokenToVerb(tok token.Token) string {
	switch tok {
	case token.EQL, token.ASSIGN:
		return "="
	default:
		// XXX return the correct QF_BV predicates. we also need type information in this function
		return tok.String()
	}
}

func verbToToken(verb string) (token.Token, bool) {
	switch verb {
	case "=":
		return token.EQL, true
	case "==":
		// XXX == only exists as a verb because we're not correctly mapping tokens to verbs
		return token.EQL, true
	case ">":
		// XXX > only exists as a verb because we're not correctly mapping tokens to verbs
		return token.GTR, true
	case "<=":
		// XXX <= only exists as a verb because we're not correctly mapping tokens to verbs
		return token.LEQ, true
	default:
		// XXX implement all the other verbs
		return 0, false
	}
}
func (b *builder) simplify0(n Node) (Node, bool) {
	switch n := n.(type) {
	case Sexp:
		changed := false

		if n.Verb == "and" {
			subst := map[Var]Node{}
			for _, in := range n.In {
				if in, ok := in.(Sexp); ok && in.Verb == "=" && len(in.In) == 2 {
					if v, ok := in.In[0].(Var); ok {
						if _, ok := subst[v]; !ok {
							subst[v] = in.In[1]
						}
					}
				}
			}

			if len(subst) > 0 {
				for i, in := range n.In {
					var ok bool
					n.In[i], ok = substitute(in, subst, true)
					log.Println(subst, "-->", in, "-->", n.In[i])
					if ok {
						changed = true
					}
				}
			}
		}

		for i, in := range n.In {
			var ok bool
			n.In[i], ok = b.simplify0(in)
			if ok {
				changed = true
			}
		}

		if len(n.In) == 2 {
			if x, ok := n.In[0].(Const); ok {
				if y, ok := n.In[1].(Const); ok {
					tok, ok := verbToToken(n.Verb)
					if ok {
						return Const{constant.MakeBool(constant.Compare(x.Value, tok, y.Value))}, true
					}
				}
			}
		}

		if len(n.In) == 2 {
			if n.Verb == "=" || n.Verb == "+" {
				if x, ok := n.In[0].(Const); ok {
					if y, ok := n.In[1].(Var); ok {
						return Op(n.Verb, y, x), true
					}
				}
			}
		}

		if n.Verb == "+" {
			if y, ok := n.In[1].(Const); ok {
				// XXX don't use string comparison
				if y.Value.ExactString() == "0" {
					return n.In[0], true
				}
			}
		}

		switch n.Verb {
		case "and":
			if len(n.In) == 0 {
				// empty and is always true
				return Const{constant.MakeBool(true)}, true
			} else if len(n.In) == 1 {
				// and with a single element is identical to that element
				return n.In[0], true
			} else {
				new := make([]Node, 0, len(n.In))
				for _, in := range n.In {
					switch in := in.(type) {
					case Const:
						if in.Value == constant.MakeBool(true) {
							// skip true elements inside and
							changed = true
						} else if in.Value == constant.MakeBool(false) {
							// the entire and is false if it contains a false element
							return in, true
						} else {
							new = append(new, in)
						}
					case Sexp:
						if in.Verb == "and" {
							// flatten nested and
							new = append(new, in.In...)
							changed = true
						} else {
							new = append(new, in)
						}
					default:
						new = append(new, in)
					}
				}

				return And(new...), changed
			}

		case "or":
			if len(n.In) == 0 {
				// empty and is always false
				return Const{constant.MakeBool(false)}, true
			} else if len(n.In) == 1 {
				// or with a single element is identical to that element
				return n.In[0], true
			} else {
				new := make([]Node, 0, len(n.In))
				for _, in := range n.In {
					switch in := in.(type) {
					case Const:
						if in.Value == constant.MakeBool(true) {
							// the entire or is true if it contains a true element
							return in, true
						} else if in.Value == constant.MakeBool(false) {
							// skip false elements inside or
							changed = true
						} else {
							new = append(new, in)
						}
					case Sexp:
						if in.Verb == "or" {
							// flatten nested or
							new = append(new, in.In...)
							changed = true
						} else {
							new = append(new, in)
						}
					default:
						new = append(new, in)
					}
				}

				return Op(n.Verb, new...), changed
			}

		case "predicate":
			// XXX this assumes that there are no loops
			pred, ok := b.predicates[n.In[0].(Var).Value]
			if ok {
				return pred, true
			} else {
				return Const{constant.MakeBool(true)}, true
			}

		default:
			return n, false
		}

	case Var:
		if k, ok := n.Value.(*ir.Const); ok {
			return Const{k.Value}, true
		} else {
			return n, false
		}

	default:
		return n, false
	}
}
