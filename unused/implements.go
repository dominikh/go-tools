package unused

import (
	"go/types"
)

// lookupMethod returns the index of and method with matching package and name, or (-1, nil).
func lookupMethod(T *types.Interface, pkg *types.Package, name string) (int, *types.Func) {
	if name != "_" {
		for i := 0; i < T.NumMethods(); i++ {
			m := T.Method(i)
			if sameId(m, pkg, name) {
				return i, m
			}
		}
	}
	return -1, nil
}

func sameId(obj types.Object, pkg *types.Package, name string) bool {
	// spec:
	// "Two identifiers are different if they are spelled differently,
	// or if they appear in different packages and are not exported.
	// Otherwise, they are the same."
	if name != obj.Name() {
		return false
	}
	// obj.Name == name
	if obj.Exported() {
		return true
	}
	// not exported, so packages must be the same (pkg == nil for
	// fields in Universe scope; this can only happen for types
	// introduced via Eval)
	if pkg == nil || obj.Pkg() == nil {
		return pkg == obj.Pkg()
	}
	// pkg != nil && obj.pkg != nil
	return pkg.Path() == obj.Pkg().Path()
}

func implements(V types.Type, T *types.Interface, msV *types.MethodSet) ([]*types.Selection, bool) {
	// fast path for common case
	if T.Empty() {
		return nil, true
	}

	if ityp, _ := V.Underlying().(*types.Interface); ityp != nil {
		// TODO(dh): is this code reachable?
		for i := 0; i < T.NumMethods(); i++ {
			m := T.Method(i)
			_, obj := lookupMethod(ityp, m.Pkg(), m.Name())
			switch {
			case obj == nil:
				return nil, false
			case !types.Identical(obj.Type(), m.Type()):
				return nil, false
			}
		}
		return nil, true
	}

	// A concrete type implements T if it implements all methods of T.
	var sels []*types.Selection
	c := newMethodChecker()
	for i := 0; i < T.NumMethods(); i++ {
		m := T.Method(i)
		sel := msV.Lookup(m.Pkg(), m.Name())
		if sel == nil {
			return nil, false
		}

		f, _ := sel.Obj().(*types.Func)
		if f == nil {
			return nil, false
		}

		if !c.satisfies(f, m) {
			return nil, false
		}

		sels = append(sels, sel)
	}
	return sels, true
}

type methodsChecker struct {
	typeParams map[*types.TypeParam]types.Type
}

func newMethodChecker() *methodsChecker {
	return &methodsChecker{
		typeParams: make(map[*types.TypeParam]types.Type),
	}
}

func (c *methodsChecker) satisfies(implFunc *types.Func, interfaceFunc *types.Func) bool {
	if types.Identical(implFunc.Type(), interfaceFunc.Type()) {
		return true
	}
	implSig, implOk := implFunc.Type().(*types.Signature)
	interfaceSig, interfaceOk := interfaceFunc.Type().(*types.Signature)
	if !implOk || !interfaceOk {
		// probably not reachable. handle conservatively.
		return false
	}

	implParams := implSig.Params()
	interfaceParams := interfaceSig.Params()
	if implParams.Len() != interfaceParams.Len() {
		return false
	}
	for i := 0; i < implParams.Len(); i++ {
		implParam := implParams.At(i)
		interfaceParam := interfaceParams.At(i)
		if types.Identical(implParam.Type(), interfaceParam.Type()) {
			continue
		}
		if tp, ok := interfaceParam.Type().(*types.TypeParam); ok {
			if c.typeParams[tp] == nil {
				if !satisfiesConstraint(implParam.Type(), tp) {
					return false
				}
				c.typeParams[tp] = implParam.Type()
				continue
			}
			if !types.Identical(c.typeParams[tp], implParam.Type()) {
				return false
			}
		}
	}
	implRess := implSig.Results()
	interfaceRess := interfaceSig.Results()
	if implRess.Len() != interfaceRess.Len() {
		return false
	}
	for i := 0; i < implRess.Len(); i++ {
		implRes := implRess.At(i)
		interfaceRes := interfaceRess.At(i)
		if types.Identical(implRes.Type(), interfaceRes.Type()) {
			continue
		}
		if tp, ok := interfaceRes.Type().(*types.TypeParam); ok {
			if c.typeParams[tp] == nil {
				if !satisfiesConstraint(implRes.Type(), tp) {
					return false
				}
				c.typeParams[tp] = implRes.Type()
				continue
			}
			if !types.Identical(c.typeParams[tp], implRes.Type()) {
				return false
			}
		}
	}
	return true
}

func satisfiesConstraint(t types.Type, tp *types.TypeParam) bool {
	bound, ok := tp.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	return types.Implements(t, bound)
}
