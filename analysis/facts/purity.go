package facts

import (
	"fmt"
	"go/types"
	"reflect"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/ir/irutil"
	"honnef.co/go/tools/internal/passes/buildir"

	"golang.org/x/tools/go/analysis"
)

const (
	// Function call produces identical outputs for identical inputs.
	// Pointers and other pointer-like types are only identical if
	// their addresses are. Strings are identical if their content
	// matches.
	Pure = iota + 1
	// Like Pure, but slices are considered equal if their contents are equal.
	SlicePure
	// Calling the function only makes sense if at least one of its outputs is consumed.
	Getter
)

type IsPure struct {
	Kind uint8
}

func (*IsPure) AFact() {}

func (d *IsPure) String() string {
	switch d.Kind {
	case Pure:
		return "is pure"
	case SlicePure:
		return "is slice pure"
	case Getter:
		return "is getter"
	default:
		return fmt.Sprintf("unknown kind of purity: %d", d.Kind)
	}
}

type PurityResult map[*types.Func]*IsPure

var Purity = &analysis.Analyzer{
	Name:       "fact_purity",
	Doc:        "Mark pure functions",
	Run:        purity,
	Requires:   []*analysis.Analyzer{buildir.Analyzer},
	FactTypes:  []analysis.Fact{(*IsPure)(nil)},
	ResultType: reflect.TypeOf(PurityResult{}),
}

func purity(pass *analysis.Pass) (interface{}, error) {
	seen := map[*ir.Function]struct{}{}
	irpkg := pass.ResultOf[buildir.Analyzer].(*buildir.IR).Pkg
	var check func(fn *ir.Function) (pure bool, kind uint8)
	check = func(fn *ir.Function) (pure bool, kind uint8) {
		if fn.Object() == nil {
			// TODO(dh): support closures
			return false, 0
		}
		var fact IsPure
		if pass.ImportObjectFact(fn.Object(), &fact) {
			return fact.Kind != 0, fact.Kind
		}
		if fn.Pkg != irpkg {
			// Function is in another package but wasn't marked as
			// pure, ergo it isn't pure
			return false, 0
		}
		// Break recursion
		if _, ok := seen[fn]; ok {
			return false, 0
		}

		seen[fn] = struct{}{}
		defer func() {
			if pure {
				pass.ExportObjectFact(fn.Object(), &IsPure{Kind: kind})
			}
		}()

		if irutil.IsStub(fn) {
			return false, 0
		}

		if kind, ok := pureStdlib[fn.Object().(*types.Func).FullName()]; ok {
			return kind != 0, kind
		}

		if fn.Signature.Results().Len() == 0 {
			// A function with no return values is empty or is doing some
			// work we cannot see (for example because of build tags);
			// don't consider it pure.
			return false, 0
		}

		for _, param := range fn.Params {
			// TODO(dh): this may not be strictly correct. pure code
			// can, to an extent, operate on non-basic types.
			if _, ok := param.Type().Underlying().(*types.Basic); !ok {
				return false, 0
			}
		}

		// Don't consider external functions pure.
		if fn.Blocks == nil {
			return false, 0
		}
		checkCall := func(common *ir.CallCommon) bool {
			if common.IsInvoke() {
				return false
			}
			builtin, ok := common.Value.(*ir.Builtin)
			if !ok {
				if common.StaticCallee() != fn {
					if common.StaticCallee() == nil {
						return false
					}
					if ok, kind := check(common.StaticCallee()); !ok || kind != Pure {
						return false
					}
				}
			} else {
				switch builtin.Name() {
				case "len", "cap":
				default:
					return false
				}
			}
			return true
		}
		for _, b := range fn.Blocks {
			for _, ins := range b.Instrs {
				switch ins := ins.(type) {
				case *ir.Call:
					if !checkCall(ins.Common()) {
						return false, 0
					}
				case *ir.Defer:
					if !checkCall(&ins.Call) {
						return false, 0
					}
				case *ir.Select:
					return false, 0
				case *ir.Send:
					return false, 0
				case *ir.Go:
					return false, 0
				case *ir.Panic:
					return false, 0
				case *ir.Store:
					return false, 0
				case *ir.FieldAddr:
					return false, 0
				case *ir.Alloc:
					return false, 0
				case *ir.Load:
					return false, 0
				}
			}
		}
		// we are only able to detect simple purity
		return true, Pure
	}
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		check(fn)
	}

	out := PurityResult{}
	for _, fact := range pass.AllObjectFacts() {
		out[fact.Object.(*types.Func)] = fact.Fact.(*IsPure)
	}
	return out, nil
}
