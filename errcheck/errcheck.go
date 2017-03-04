package errcheck

import (
	"go/token"
	"go/types"

	"honnef.co/go/tools/functions"
	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/ssa"
	"honnef.co/go/tools/ssa/ssautil"
)

type Checker struct {
	funcDescs *functions.Descriptions
	funcs     map[*token.File][]*ssa.Function
}

func NewChecker() *Checker {
	return &Checker{}
}

func (c *Checker) Funcs() map[string]lint.Func {
	return map[string]lint.Func{
		"ERR1000": c.CheckErrcheck,
	}
}

func (c *Checker) funcsForFile(f *lint.File) []*ssa.Function {
	return c.funcs[f.Program.Fset.File(f.File.Pos())]
}

func (c *Checker) Init(prog *lint.Program) {
	c.funcs = map[*token.File][]*ssa.Function{}
	c.funcDescs = functions.NewDescriptions(prog.SSA)

	fns := ssautil.AllFunctions(prog.SSA)

	for fn := range fns {
		if fn.Blocks == nil {
			continue
		}
		if fn.Synthetic != "" && (fn.Package() == nil || fn != fn.Package().Members["init"]) {
			// Don't track synthetic functions, unless they're the
			// init function
			continue
		}
		pos := fn.Pos()
		if pos == 0 {
			for _, pkg := range prog.Packages {
				if pkg.SSAPkg == fn.Pkg {
					pos = pkg.PkgInfo.Files[0].Pos()
					break
				}
			}
		}
		f := prog.Prog.Fset.File(pos)
		c.funcs[f] = append(c.funcs[f], fn)
	}
}

func (c *Checker) CheckErrcheck(f *lint.File) {
	for _, ssafn := range c.funcsForFile(f) {
		for _, b := range ssafn.Blocks {
			for _, ins := range b.Instrs {
				ssacall, ok := ins.(ssa.CallInstruction)
				if !ok {
					continue
				}

				switch lint.CallName(ssacall.Common()) {
				case "fmt.Print", "fmt.Println", "fmt.Printf":
					continue
				}
				isRecover := false
				if builtin, ok := ssacall.Common().Value.(*ssa.Builtin); ok {
					isRecover = ok && builtin.Name() == "recover"
				}

				switch ins := ins.(type) {
				case ssa.Value:
					refs := ins.Referrers()
					if refs == nil || len(lint.FilterDebug(*refs)) != 0 {
						continue
					}
				case ssa.Instruction:
					// will be a 'go' or 'defer', neither of which has usable return values
				default:
					// shouldn't happen
					continue
				}

				if ssacall.Common().IsInvoke() {
					if sc, ok := ssacall.Common().Value.(*ssa.Call); ok {
						// TODO(dh): support multiple levels of
						// interfaces, not just one
						ssafn := sc.Common().StaticCallee()
						if ssafn != nil {
							ct := c.funcDescs.Get(ssafn).ConcreteReturnTypes
							// TODO(dh): support >1 concrete types
							if ct != nil && len(ct) == 1 {
								// TODO(dh): do we have access to a
								// cached method set somewhere?
								ms := types.NewMethodSet(ct[0].At(ct[0].Len() - 1).Type())
								// TODO(dh): where can we get the pkg
								// for Lookup? Passing nil works fine
								// for exported methods, but will fail
								// on unexported ones
								// TODO(dh): holy nesting and poor
								// variable names, clean this up
								fn, _ := ms.Lookup(nil, ssacall.Common().Method.Name()).Obj().(*types.Func)
								if fn != nil {
									ssafn := f.Pkg.SSAPkg.Prog.FuncValue(fn)
									if ssafn != nil {
										if c.funcDescs.Get(ssafn).NilError {
											continue
										}
									}
								}
							}
						}
					}
				} else {
					ssafn := ssacall.Common().StaticCallee()
					if ssafn != nil {
						if c.funcDescs.Get(ssafn).NilError {
							// Don't complain when the error is known to be nil
							continue
						}
					}
				}
				switch lint.CallName(ssacall.Common()) {
				case "(*os.File).Close":
					recv := ssacall.Common().Args[0]
					if isReadOnlyFile(recv, nil) {
						continue
					}
				}

				res := ssacall.Common().Signature().Results()
				if res.Len() == 0 {
					continue
				}
				if !isRecover {
					last := res.At(res.Len() - 1)
					if types.TypeString(last.Type(), nil) != "error" {
						continue
					}
				}
				f.Errorf(ins, "unchecked error")
			}
		}
	}
}

func isReadOnlyFile(val ssa.Value, seen map[ssa.Value]bool) bool {
	if seen == nil {
		seen = map[ssa.Value]bool{}
	}
	if seen[val] {
		return true
	}
	seen[val] = true
	switch val := val.(type) {
	case *ssa.Phi:
		for _, edge := range val.Edges {
			if !isReadOnlyFile(edge, seen) {
				return false
			}
		}
		return true
	case *ssa.Extract:
		call, ok := val.Tuple.(*ssa.Call)
		if !ok {
			return false
		}
		switch lint.CallName(call.Common()) {
		case "os.Open":
			return true
		case "os.OpenFile":
			flags, ok := call.Common().Args[1].(*ssa.Const)
			return ok && flags.Uint64() == 0
		}
		return false
	}
	return false
}
