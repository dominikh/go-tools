package smt

// XXX consider the interaction of path predicates and equality propagation

import (
	"fmt"
	"go/constant"
)

type Node interface {
	String() string
	Equal(Node) bool
}

type Var struct {
	// XXX use integers, not strings
	Name string
}

func (v Var) String() string {
	return v.Name
}

func (v Var) Equal(o Node) bool {
	return any(v) == o
}

type Const struct {
	Value constant.Value
}

func (c Const) String() string {
	return c.Value.ExactString()
}

func (c Const) Equal(o Node) bool {
	return any(c) == o
}

type Sexp struct {
	// OPT don't use string for Verb
	Verb string
	// TODO we need 3 arguments because of ITE. Fix that by translating ITE down to ands/ors
	In [3]Node
}

func (s Sexp) Equal(o Node) bool {
	if s == o {
		return true
	}
	so, ok := o.(Sexp)
	if !ok {
		return false
	}
	for i := range s.In {
		if !s.In[i].Equal(so.In[i]) {
			return false
		}
	}
	return true
}

func (s Sexp) String() string {
	if s.In[2] == nil {
		if s.In[1] == nil {
			return fmt.Sprintf("(%s %s)", s.Verb, s.In[0])
		} else {
			return fmt.Sprintf("(%s %s %s)", s.Verb, s.In[0], s.In[1])
		}
	} else {
		return fmt.Sprintf("(%s %s %s %s)", s.Verb, s.In[0], s.In[1], s.In[2])
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
		and := Op("and", nodes[0], nodes[1])
		for _, n := range nodes[2:] {
			and = Op("and", n, and)
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
		or := Op("or", nodes[0], nodes[1])
		for _, n := range nodes[2:] {
			or = Op("or", n, or)
		}
		return or
	}
}

func Xor(a, b Node) Node {
	return Op("xor", a, b)
}

func Equal(a, b Node) Node {
	return Op("=", a, b)
}

func Not(a Node) Node {
	return Op("not", a, nil)
}

func Op(verb string, a, b Node) Node {
	return Sexp{
		Verb: verb,
		In:   [3]Node{a, b, nil},
	}
}

func ITE(cond Node, t Node, f Node) Sexp {
	return Sexp{
		Verb: "ite",
		In:   [3]Node{cond, t, f},
	}
}

// Our formulas are generally small enough that we don't care about on-the-fly simplifications. Instead use a simple
// fixpoint approach.

/*
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
	case *Sexp:
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
	case *Sexp:
		changed := false

		if n.Verb == "and" {
			subst := map[Var]Node{}
			for _, in := range n.In {
				if in, ok := in.(*Sexp); ok && in.Verb == "=" && len(in.In) == 2 {
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
					case *Sexp:
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

				return b.And(new...), changed
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
					case *Sexp:
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

func Dot(n Node) string {
	var dfs func(Node) string
	seen := map[Node]struct{}{}
	var out string
	dfs = func(n Node) string {
		switch n := n.(type) {
		case Raw:
			return string(n)
		case Const:
			return n.Value.ExactString()
		case Var:
			return n.Value.Name()
		case *Sexp:
			if _, ok := seen[n]; !ok {
				seen[n] = struct{}{}
				out += fmt.Sprintf("sexp%p [label=%q]\n", n, n.Verb)
				for _, in := range n.In {
					out += fmt.Sprintf("%s -> sexp%p\n", dfs(in), n)
				}
			}
			return fmt.Sprintf("sexp%p", n)
		default:
			panic(fmt.Sprintf("%T", n))
		}
	}

	dfs(n)

	return out
}
*/
