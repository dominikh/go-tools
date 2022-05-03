package smt

// XXX consider the interaction of path predicates and equality propagation

import (
	"fmt"
	"go/constant"
	"strings"

	"honnef.co/go/tools/go/ir"
)

type Sexp struct {
	// OPT don't use string for Verb
	Verb     string
	Value    string         // when Verb == var
	Constant constant.Value // when Verb == const
	In       []*Sexp        // for all other verbs
}

func (s *Sexp) Equal(o *Sexp) bool {
	if s == o {
		return true
	}
	if s.Verb != o.Verb || s.Value != o.Value || s.Constant != o.Constant || len(s.In) != len(o.In) {
		return false
	}
	for i := range s.In {
		if !s.In[i].Equal(o.In[i]) {
			return false
		}
	}
	return true
}

func (s *Sexp) String() string {
	switch s.Verb {
	case "var":
		if len(s.In) != 0 {
			panic("XXX")
		}
		return fmt.Sprintf("(var %s)", s.Value)
	case "const":
		if len(s.In) != 0 {
			panic("XXX")
		}
		return fmt.Sprintf("(const %s)", s.Constant)
	default:
		args := make([]string, len(s.In))
		for i, arg := range s.In {
			args[i] = fmt.Sprintf("%s", arg)
		}
		return fmt.Sprintf("(%s %s)", s.Verb, strings.Join(args, " "))
	}

}

type key [2]any

type builder struct {
	vars       map[ir.Value]*Sexp
	predicates map[ir.Value]*Sexp

	sexps map[sexpKey]*Sexp
}

type sexpKey struct {
	verb string
	args [2]any
}

func (bl *builder) dedup(n *Sexp) *Sexp {
	// XXX support sexps with any number of inputs
	// XXX make sure that every code path that optimizes sexps re-dedups

	if len(n.In) != 2 {
		return n
	}
	key := sexpKey{n.Verb, [2]any{n.In[0], n.In[1]}}
	if dup, ok := bl.sexps[key]; ok {
		return dup
	} else {
		bl.sexps[key] = n
		return n
	}
}

// TODO number nodes for a canonical ordering

func (b *builder) value(v ir.Value, n *Sexp) {
	b.vars[v] = n
}

func (b *builder) predicate(v ir.Value, p *Sexp) {
	b.predicates[v] = p
}

func (bl *builder) And(nodes ...*Sexp) *Sexp {
	return bl.dedup(Op("and", nodes...))
}

func (bl *builder) Or(nodes ...*Sexp) *Sexp {
	return bl.dedup(Op("or", nodes...))
}

func (bl *builder) Xor(nodes ...*Sexp) *Sexp {
	return bl.dedup(Op("xor", nodes...))
}

func Equal(a, b *Sexp) *Sexp {
	return Op("=", a, b)
}

func Not(a *Sexp) *Sexp {
	return Op("not", a)
}

func Op(verb string, nodes ...*Sexp) *Sexp {
	if verb == "var" || verb == "const" {
		panic("XXX")
	}
	return &Sexp{
		Verb: verb,
		In:   nodes,
	}
}

func Var(v string) *Sexp {
	return &Sexp{Verb: "var", Value: v}
}

func Const(c constant.Value) *Sexp {
	return &Sexp{Verb: "const", Constant: c}
}

func ITE(cond *Sexp, t *Sexp, f *Sexp) *Sexp {
	return Op("ite", cond, t, f)
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
