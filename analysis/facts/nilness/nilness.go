package nilness

import (
	"fmt"
	"go/token"
	"go/types"
	"reflect"

	"honnef.co/go/tools/go/ir"
	"honnef.co/go/tools/go/types/typeutil"
	"honnef.co/go/tools/internal/passes/buildir"

	"golang.org/x/tools/go/analysis"
)

// neverReturnsNilFact denotes that a function's return value will never
// be nil (typed or untyped). The analysis errs on the side of false
// negatives.
type neverReturnsNilFact struct {
	Rets uint8
}

func (*neverReturnsNilFact) AFact() {}
func (fact *neverReturnsNilFact) String() string {
	return fmt.Sprintf("never returns nil: %08b", fact.Rets)
}

type Result struct {
	m map[*types.Func]uint8
}

var Analysis = &analysis.Analyzer{
	Name:       "nilness",
	Doc:        "Annotates return values that will never be nil (typed or untyped)",
	Run:        run,
	Requires:   []*analysis.Analyzer{buildir.Analyzer},
	FactTypes:  []analysis.Fact{(*neverReturnsNilFact)(nil)},
	ResultType: reflect.TypeOf((*Result)(nil)),
}

// MayReturnNil reports whether the ret's return value of fn might be
// a typed or untyped nil value. The value of ret is zero-based.
//
// The analysis has false positives: MayReturnNil can incorrectly
// report true, but never incorrectly reports false.
func (r *Result) MayReturnNil(fn *types.Func, ret int) bool {
	if !typeutil.IsPointerLike(fn.Type().(*types.Signature).Results().At(ret).Type()) {
		return false
	}
	return (r.m[fn] & (1 << ret)) == 0
}

func run(pass *analysis.Pass) (interface{}, error) {
	seen := map[*ir.Function]struct{}{}
	out := &Result{
		m: map[*types.Func]uint8{},
	}
	for _, fn := range pass.ResultOf[buildir.Analyzer].(*buildir.IR).SrcFuncs {
		impl(pass, fn, seen)
	}

	for _, fact := range pass.AllObjectFacts() {
		out.m[fact.Object.(*types.Func)] = fact.Fact.(*neverReturnsNilFact).Rets
	}

	return out, nil
}

func impl(pass *analysis.Pass, fn *ir.Function, seenFns map[*ir.Function]struct{}) (out uint8) {
	if fn.Signature.Results().Len() > 8 {
		return 0
	}
	if fn.Object() == nil {
		// TODO(dh): support closures
		return 0
	}
	if fact := new(neverReturnsNilFact); pass.ImportObjectFact(fn.Object(), fact) {
		return fact.Rets
	}
	if fn.Pkg != pass.ResultOf[buildir.Analyzer].(*buildir.IR).Pkg {
		return 0
	}
	if fn.Blocks == nil {
		return 0
	}
	if _, ok := seenFns[fn]; ok {
		// break recursion
		return 0
	}

	seenFns[fn] = struct{}{}
	defer func() {
		for i := 0; i < fn.Signature.Results().Len(); i++ {
			if !typeutil.IsPointerLike(fn.Signature.Results().At(i).Type()) {
				// we don't need facts to know that non-pointer types
				// can't be nil. zeroing out those bits may result in
				// all bits being zero, in which case we don't have to
				// save any fact.
				out &= ^(1 << i)
			}
		}
		if out > 0 {
			pass.ExportObjectFact(fn.Object(), &neverReturnsNilFact{out})
		}
	}()

	seen := map[ir.Value]struct{}{}
	var mightReturnNil func(v ir.Value) bool
	mightReturnNil = func(v ir.Value) bool {
		if _, ok := seen[v]; ok {
			// break cycle
			return true
		}
		if !typeutil.IsPointerLike(v.Type()) {
			return false
		}
		seen[v] = struct{}{}
		switch v := v.(type) {
		case *ir.MakeInterface:
			return mightReturnNil(v.X)
		case *ir.Convert:
			return mightReturnNil(v.X)
		case *ir.Slice:
			return mightReturnNil(v.X)
		case *ir.Phi:
			for _, e := range v.Edges {
				if mightReturnNil(e) {
					return true
				}
			}
			return false
		case *ir.Extract:
			switch d := v.Tuple.(type) {
			case *ir.Call:
				if callee := d.Call.StaticCallee(); callee != nil {
					return impl(pass, callee, seenFns)&(1<<v.Index) == 0
				} else {
					return true
				}
			case *ir.TypeAssert, *ir.Next, *ir.Select, *ir.MapLookup, *ir.TypeSwitch, *ir.Recv:
				// we don't need to look at the Extract's index
				// because we've already checked its type.
				return true
			default:
				panic(fmt.Sprintf("internal error: unhandled type %T", d))
			}
		case *ir.Call:
			if callee := v.Call.StaticCallee(); callee != nil {
				ret := impl(pass, callee, seenFns)
				return ret&1 == 0
			} else {
				return true
			}
		case *ir.BinOp, *ir.UnOp, *ir.Alloc, *ir.FieldAddr, *ir.IndexAddr, *ir.Global, *ir.MakeSlice, *ir.MakeClosure, *ir.Function, *ir.MakeMap, *ir.MakeChan:
			return false
		case *ir.Sigma:
			iff, ok := v.From.Control().(*ir.If)
			if !ok {
				return true
			}
			binop, ok := iff.Cond.(*ir.BinOp)
			if !ok {
				return true
			}
			isNil := func(v ir.Value) bool {
				k, ok := v.(*ir.Const)
				if !ok {
					return false
				}
				return k.Value == nil
			}
			if binop.X == v.X && isNil(binop.Y) || binop.Y == v.X && isNil(binop.X) {
				op := binop.Op
				if v.From.Succs[0] != v.Block() {
					// we're in the false branch, negate op
					switch op {
					case token.EQL:
						op = token.NEQ
					case token.NEQ:
						op = token.EQL
					default:
						panic(fmt.Sprintf("internal error: unhandled token %v", op))
					}
				}
				switch op {
				case token.EQL:
					return true
				case token.NEQ:
					return false
				default:
					panic(fmt.Sprintf("internal error: unhandled token %v", op))
				}
			}
			return true
		case *ir.ChangeType:
			return mightReturnNil(v.X)
		case *ir.TypeAssert, *ir.ChangeInterface, *ir.Field, *ir.Const, *ir.Index, *ir.MapLookup, *ir.Parameter, *ir.Load, *ir.Recv, *ir.TypeSwitch:
			return true
		default:
			panic(fmt.Sprintf("internal error: unhandled type %T", v))
		}
	}
	ret := fn.Exit.Control().(*ir.Return)
	for i, v := range ret.Results {
		if !mightReturnNil(v) {
			out |= 1 << i
		}
	}
	return out
}
