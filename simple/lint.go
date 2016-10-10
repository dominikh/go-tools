// Package simple contains a linter for Go source code.
package simple // import "honnef.co/go/simple"

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"reflect"
	"strconv"
	"strings"

	"honnef.co/go/lint"
)

var Funcs = []lint.Func{
	LintSingleCaseSelect,
	LintLoopCopy,
	LintIfBoolCmp,
	LintStringsContains,
	LintBytesCompare,
	LintRanges,
	LintForTrue,
	LintRegexpRaw,
	LintIfReturn,
	LintRedundantNilCheckWithLen,
	LintSlicing,
	LintLoopAppend,
	LintTimeSince,
	LintSimplerReturn,
	LintReceiveIntoBlank,
	LintFormatInt,
	LintSimplerStructConversion,
	LintTrim,
}

func LintSingleCaseSelect(f *lint.File) {
	isSingleSelect := func(node ast.Node) bool {
		v, ok := node.(*ast.SelectStmt)
		if !ok {
			return false
		}
		return len(v.Body.List) == 1
	}

	seen := map[ast.Node]struct{}{}
	f.Walk(func(node ast.Node) bool {
		switch v := node.(type) {
		case *ast.ForStmt:
			if len(v.Body.List) != 1 {
				return true
			}
			if !isSingleSelect(v.Body.List[0]) {
				return true
			}
			if _, ok := v.Body.List[0].(*ast.SelectStmt).Body.List[0].(*ast.CommClause).Comm.(*ast.SendStmt); ok {
				// Don't suggest using range for channel sends
				return true
			}
			seen[v.Body.List[0]] = struct{}{}
			f.Errorf(node, 1, lint.Category("range-loop"), "should use for range instead of for { select {} }")
		case *ast.SelectStmt:
			if _, ok := seen[v]; ok {
				return true
			}
			if !isSingleSelect(v) {
				return true
			}
			f.Errorf(node, 1, lint.Category("FIXME"), "should use a simple channel send/receive instead of select with a single case")
			return true
		}
		return true
	})
}

func LintLoopCopy(f *lint.File) {
	fn := func(node ast.Node) bool {
		loop, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}

		if loop.Key == nil {
			return true
		}
		if len(loop.Body.List) != 1 {
			return true
		}
		stmt, ok := loop.Body.List[0].(*ast.AssignStmt)
		if !ok {
			return true
		}
		if stmt.Tok != token.ASSIGN || len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
			return true
		}
		lhs, ok := stmt.Lhs[0].(*ast.IndexExpr)
		if !ok {
			return true
		}
		if _, ok := f.Pkg.TypesInfo.TypeOf(lhs.X).(*types.Slice); !ok {
			return true
		}
		lidx, ok := lhs.Index.(*ast.Ident)
		if !ok {
			return true
		}
		key, ok := loop.Key.(*ast.Ident)
		if !ok {
			return true
		}
		if f.Pkg.TypesInfo.TypeOf(lhs) == nil || f.Pkg.TypesInfo.TypeOf(stmt.Rhs[0]) == nil {
			return true
		}
		if f.Pkg.TypesInfo.ObjectOf(lidx) != f.Pkg.TypesInfo.ObjectOf(key) {
			return true
		}
		if !types.Identical(f.Pkg.TypesInfo.TypeOf(lhs), f.Pkg.TypesInfo.TypeOf(stmt.Rhs[0])) {
			return true
		}
		if _, ok := f.Pkg.TypesInfo.TypeOf(loop.X).(*types.Slice); !ok {
			return true
		}

		if rhs, ok := stmt.Rhs[0].(*ast.IndexExpr); ok {
			rx, ok := rhs.X.(*ast.Ident)
			_ = rx
			if !ok {
				return true
			}
			ridx, ok := rhs.Index.(*ast.Ident)
			if !ok {
				return true
			}
			if f.Pkg.TypesInfo.ObjectOf(ridx) != f.Pkg.TypesInfo.ObjectOf(key) {
				return true
			}
		} else if rhs, ok := stmt.Rhs[0].(*ast.Ident); ok {
			value, ok := loop.Value.(*ast.Ident)
			if !ok {
				return true
			}
			if f.Pkg.TypesInfo.ObjectOf(rhs) != f.Pkg.TypesInfo.ObjectOf(value) {
				return true
			}
		} else {
			return true
		}
		f.Errorf(loop, 1, lint.Category("FIXME"), "should use copy() instead of a loop")
		return true
	}
	f.Walk(fn)
}

func LintIfBoolCmp(f *lint.File) {
	fn := func(node ast.Node) bool {
		expr, ok := node.(*ast.BinaryExpr)
		if !ok || (expr.Op != token.EQL && expr.Op != token.NEQ) {
			return true
		}
		x := f.IsBoolConst(expr.X)
		y := f.IsBoolConst(expr.Y)
		if x || y {
			var other ast.Expr
			var val bool
			if x {
				val = f.BoolConst(expr.X)
				other = expr.Y
			} else {
				val = f.BoolConst(expr.Y)
				other = expr.X
			}
			op := ""
			if (expr.Op == token.EQL && !val) || (expr.Op == token.NEQ && val) {
				op = "!"
			}
			f.Errorf(expr, 1, lint.Category("FIXME"), "should omit comparison to bool constant, can be simplified to %s%s",
				op, f.Render(other))
		}
		return true
	}
	f.Walk(fn)
}

func LintStringsContains(f *lint.File) {
	// map of value to token to bool value
	allowed := map[string]map[token.Token]bool{
		"-1": {token.GTR: true, token.NEQ: true, token.EQL: false},
		"0":  {token.GEQ: true, token.LSS: false},
	}
	fn := func(node ast.Node) bool {
		expr, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		switch expr.Op {
		case token.GEQ, token.GTR, token.NEQ, token.LSS, token.EQL:
		default:
			return true
		}

		value, ok := lint.ExprToInt(expr.Y)
		if !ok {
			return true
		}

		allowedOps, ok := allowed[value]
		if !ok {
			return true
		}
		b, ok := allowedOps[expr.Op]
		if !ok {
			return true
		}

		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		funIdent := sel.Sel
		if pkgIdent.Name != "strings" && pkgIdent.Name != "bytes" {
			return true
		}
		newFunc := ""
		switch funIdent.Name {
		case "IndexRune":
			newFunc = "ContainsRune"
		case "IndexAny":
			newFunc = "ContainsAny"
		case "Index":
			newFunc = "Contains"
		default:
			return true
		}

		prefix := ""
		if !b {
			prefix = "!"
		}
		f.Errorf(node, 1, "should use %s%s.%s(%s) instead", prefix, pkgIdent.Name, newFunc, f.RenderArgs(call.Args))

		return true
	}
	f.Walk(fn)
}

func LintBytesCompare(f *lint.File) {
	fn := func(node ast.Node) bool {
		expr, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		if expr.Op != token.NEQ && expr.Op != token.EQL {
			return true
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !lint.IsPkgDot(call.Fun, "bytes", "Compare") {
			return true
		}
		value, ok := lint.ExprToInt(expr.Y)
		if !ok {
			return true
		}
		if value != "0" {
			return true
		}
		args := f.RenderArgs(call.Args)
		prefix := ""
		if expr.Op == token.NEQ {
			prefix = "!"
		}
		f.Errorf(node, 1, lint.Category("FIXME"), "should use %sbytes.Equal(%s) instead", prefix, args)
		return true
	}
	f.Walk(fn)
}

func LintRanges(f *lint.File) {
	f.Walk(func(node ast.Node) bool {
		rs, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}
		if lint.IsIdent(rs.Key, "_") && (rs.Value == nil || lint.IsIdent(rs.Value, "_")) {
			f.Errorf(rs.Key, 1, lint.Category("range-loop"), "should omit values from range; this loop is equivalent to `for range ...`")
		}

		return true
	})
}

func LintForTrue(f *lint.File) {
	fn := func(node ast.Node) bool {
		loop, ok := node.(*ast.ForStmt)
		if !ok {
			return true
		}
		if loop.Init != nil || loop.Post != nil {
			return true
		}
		if !f.IsBoolConst(loop.Cond) || !f.BoolConst(loop.Cond) {
			return true
		}
		f.Errorf(loop, 1, lint.Category("FIXME"), "should use for {} instead of for true {}")
		return true
	}
	f.Walk(fn)
}

func LintRegexpRaw(f *lint.File) {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !lint.IsPkgDot(call.Fun, "regexp", "MustCompile") && !lint.IsPkgDot(call.Fun, "regexp", "Compile") {
			return true
		}
		if len(call.Args) != 1 {
			// invalid function call
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			// TODO(dominikh): support string concat, maybe support constants
			return true
		}
		if lit.Kind != token.STRING {
			// invalid function call
			return true
		}
		if f.Src[f.Fset.Position(lit.Pos()).Offset] != '"' {
			// already a raw string
			return true
		}
		val := lit.Value
		if !strings.Contains(val, `\\`) {
			return true
		}

		bs := false
		for _, c := range val {
			if !bs && c == '\\' {
				bs = true
				continue
			}
			if bs && c == '\\' {
				bs = false
				continue
			}
			if bs {
				// backslash followed by non-backslash -> escape sequence
				return true
			}
		}

		f.Errorf(call, 1, lint.Category("FIXME"), "should use raw string (`...`) with regexp.%s to avoid having to escape twice", sel.Sel.Name)
		return true
	}
	f.Walk(fn)
}

func LintIfReturn(f *lint.File) {
	fn := func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		l := len(block.List)
		if l < 2 {
			return true
		}
		n1, n2 := block.List[l-2], block.List[l-1]

		if len(block.List) >= 3 {
			if _, ok := block.List[l-3].(*ast.IfStmt); ok {
				// Do not flag a series of if statements
				return true
			}
		}
		// if statement with no init, no else, a single condition
		// checking an identifier or function call and just a return
		// statement in the body, that returns a boolean constant
		ifs, ok := n1.(*ast.IfStmt)
		if !ok {
			return true
		}
		if ifs.Else != nil || ifs.Init != nil {
			return true
		}
		if len(ifs.Body.List) != 1 {
			return true
		}
		if op, ok := ifs.Cond.(*ast.BinaryExpr); ok {
			switch op.Op {
			case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
			default:
				return true
			}
		}
		ret1, ok := ifs.Body.List[0].(*ast.ReturnStmt)
		if !ok {
			return true
		}
		if len(ret1.Results) != 1 {
			return true
		}
		if !f.IsBoolConst(ret1.Results[0]) {
			return true
		}

		ret2, ok := n2.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		if len(ret2.Results) != 1 {
			return true
		}
		if !f.IsBoolConst(ret2.Results[0]) {
			return true
		}
		f.Errorf(n1, 1, lint.Category("FIXME"), "should use 'return <expr>' instead of 'if <expr> { return <bool> }; return <bool>'")
		return true
	}
	f.Walk(fn)
}

// LintRedundantNilCheckWithLen checks for the following reduntant nil-checks:
//
//   if x == nil || len(x) == 0 {}
//   if x != nil && len(x) != 0 {}
//   if x != nil && len(x) == N {} (where N != 0)
//   if x != nil && len(x) > N {}
//   if x != nil && len(x) >= N {} (where N != 0)
//
func LintRedundantNilCheckWithLen(f *lint.File) {
	isConstZero := func(expr ast.Expr) (isConst bool, isZero bool) {
		lit, ok := expr.(*ast.BasicLit)
		if ok {
			return true, lit.Kind == token.INT && lit.Value == "0"
		}
		id, ok := expr.(*ast.Ident)
		if !ok {
			return false, false
		}
		c, ok := f.Pkg.TypesInfo.ObjectOf(id).(*types.Const)
		if !ok {
			return false, false
		}
		return true, c.Val().Kind() == constant.Int && c.Val().String() == "0"
	}

	fn := func(node ast.Node) bool {
		// check that expr is "x || y" or "x && y"
		expr, ok := node.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		if expr.Op != token.LOR && expr.Op != token.LAND {
			return true
		}
		eqNil := expr.Op == token.LOR

		// check that x is "xx == nil" or "xx != nil"
		x, ok := expr.X.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		if eqNil && x.Op != token.EQL {
			return true
		}
		if !eqNil && x.Op != token.NEQ {
			return true
		}
		xx, ok := x.X.(*ast.Ident)
		if !ok {
			return true
		}
		if !lint.IsNil(x.Y) {
			return true
		}

		// check that y is "len(xx) == 0" or "len(xx) ... "
		y, ok := expr.Y.(*ast.BinaryExpr)
		if !ok {
			return true
		}
		if eqNil && y.Op != token.EQL { // must be len(xx) *==* 0
			return false
		}
		yx, ok := y.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		yxFun, ok := yx.Fun.(*ast.Ident)
		if !ok || yxFun.Name != "len" || len(yx.Args) != 1 {
			return true
		}
		yxArg, ok := yx.Args[0].(*ast.Ident)
		if !ok {
			return true
		}
		if yxArg.Name != xx.Name {
			return true
		}

		if eqNil && !lint.IsZero(y.Y) { // must be len(x) == *0*
			return true
		}

		if !eqNil {
			isConst, isZero := isConstZero(y.Y)
			if !isConst {
				return true
			}
			switch y.Op {
			case token.EQL:
				// avoid false positive for "xx != nil && len(xx) == 0"
				if isZero {
					return true
				}
			case token.GEQ:
				// avoid false positive for "xx != nil && len(xx) >= 0"
				if isZero {
					return true
				}
			case token.NEQ:
				// avoid false positive for "xx != nil && len(xx) != <non-zero>"
				if !isZero {
					return true
				}
			case token.GTR:
				// ok
			default:
				return true
			}
		}

		// finally check that xx type is one of array, slice, map or chan
		// this is to prevent false positive in case if xx is a pointer to an array
		var nilType string
		switch f.Pkg.TypesInfo.TypeOf(xx).(type) {
		case *types.Slice:
			nilType = "nil slices"
		case *types.Map:
			nilType = "nil maps"
		case *types.Chan:
			nilType = "nil channels"
		default:
			return true
		}
		f.Errorf(expr, 1, lint.Category("FIXME"), "should omit nil check; len() for %s is defined as zero", nilType)
		return true
	}
	f.Walk(fn)
}

func LintSlicing(f *lint.File) {
	fn := func(node ast.Node) bool {
		n, ok := node.(*ast.SliceExpr)
		if !ok {
			return true
		}
		if n.Max != nil {
			return true
		}
		s, ok := n.X.(*ast.Ident)
		if !ok || s.Obj == nil {
			return true
		}
		call, ok := n.High.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
			return true
		}
		fun, ok := call.Fun.(*ast.Ident)
		if !ok || fun.Name != "len" {
			return true
		}
		if _, ok := f.Pkg.TypesInfo.ObjectOf(fun).(*types.Builtin); !ok {
			return true
		}
		arg, ok := call.Args[0].(*ast.Ident)
		if !ok || arg.Obj != s.Obj {
			return true
		}
		f.Errorf(n, 1, "should omit second index in slice, s[a:len(s)] is identical to s[a:]")
		return true
	}
	f.Walk(fn)
}

func refersTo(info *types.Info, expr ast.Expr, ident *ast.Ident) bool {
	found := false
	fn := func(node ast.Node) bool {
		ident2, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if info.ObjectOf(ident) == info.ObjectOf(ident2) {
			found = true
			return false
		}
		return true
	}
	ast.Inspect(expr, fn)
	return found
}

func LintLoopAppend(f *lint.File) {
	fn := func(node ast.Node) bool {
		loop, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}
		if !lint.IsBlank(loop.Key) {
			return true
		}
		val, ok := loop.Value.(*ast.Ident)
		if !ok {
			return true
		}
		if len(loop.Body.List) != 1 {
			return true
		}
		stmt, ok := loop.Body.List[0].(*ast.AssignStmt)
		if !ok {
			return true
		}
		if stmt.Tok != token.ASSIGN || len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
			return true
		}
		if refersTo(f.Pkg.TypesInfo, stmt.Lhs[0], val) {
			return true
		}
		call, ok := stmt.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		if len(call.Args) != 2 || call.Ellipsis.IsValid() {
			return true
		}
		fun, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}
		obj := f.Pkg.TypesInfo.ObjectOf(fun)
		fn, ok := obj.(*types.Builtin)
		if !ok || fn.Name() != "append" {
			return true
		}

		src := f.Pkg.TypesInfo.TypeOf(loop.X)
		dst := f.Pkg.TypesInfo.TypeOf(call.Args[0])
		// TODO(dominikh) remove nil check once Go issue #15173 has
		// been fixed
		if src == nil {
			return true
		}
		if !types.Identical(src, dst) {
			return true
		}

		if f.Render(stmt.Lhs[0]) != f.Render(call.Args[0]) {
			return true
		}

		el, ok := call.Args[1].(*ast.Ident)
		if !ok {
			return true
		}
		if f.Pkg.TypesInfo.ObjectOf(val) != f.Pkg.TypesInfo.ObjectOf(el) {
			return true
		}
		f.Errorf(loop, 1, "should replace loop with %s = append(%s, %s...)",
			f.Render(stmt.Lhs[0]), f.Render(call.Args[0]), f.Render(loop.X))
		return true
	}
	f.Walk(fn)
}

func LintTimeSince(f *lint.File) {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		subcall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !lint.IsPkgDot(subcall.Fun, "time", "Now") {
			return true
		}
		if sel.Sel.Name != "Sub" {
			return true
		}
		f.Errorf(call, 1, "should use time.Since instead of time.Now().Sub")
		return true
	}
	f.Walk(fn)
}

func LintSimplerReturn(f *lint.File) {
	fn1 := func(node ast.Node) bool {
		var ret *ast.FieldList
		switch x := node.(type) {
		case *ast.FuncDecl:
			ret = x.Type.Results
		case *ast.FuncLit:
			ret = x.Type.Results
		default:
			return true
		}
		if ret == nil {
			return true
		}

		fn2 := func(node ast.Node) bool {
			block, ok := node.(*ast.BlockStmt)
			if !ok {
				return true
			}
			if len(block.List) < 2 {
				return true
			}

		outer:
			for i, stmt := range block.List {
				if i == len(block.List)-1 {
					break
				}
				if i > 0 {
					// don't flag an if in a series of ifs
					if _, ok := block.List[i-1].(*ast.IfStmt); ok {
						continue
					}
				}

				// if <id1> != nil
				ifs, ok := stmt.(*ast.IfStmt)
				if !ok || len(ifs.Body.List) != 1 || ifs.Else != nil {
					continue
				}
				expr, ok := ifs.Cond.(*ast.BinaryExpr)
				if !ok || expr.Op != token.NEQ || !lint.IsNil(expr.Y) {
					continue
				}
				id1, ok := expr.X.(*ast.Ident)
				if !ok {
					continue
				}

				// return ..., <id1>
				ret1, ok := ifs.Body.List[0].(*ast.ReturnStmt)
				if !ok || len(ret1.Results) == 0 {
					continue
				}
				var results1 []types.Object
				for _, res := range ret1.Results {
					ident, ok := res.(*ast.Ident)
					if !ok {
						continue outer
					}
					results1 = append(results1, f.Pkg.TypesInfo.ObjectOf(ident))
				}
				if results1[len(results1)-1] != f.Pkg.TypesInfo.ObjectOf(id1) {
					continue
				}

				// return ..., [<id1> | nil]
				ret2, ok := block.List[i+1].(*ast.ReturnStmt)
				if !ok || len(ret2.Results) == 0 {
					continue
				}
				var results2 []types.Object
				for _, res := range ret2.Results {
					ident, ok := res.(*ast.Ident)
					if !ok {
						continue outer
					}
					results2 = append(results2, f.Pkg.TypesInfo.ObjectOf(ident))
				}
				_, isNil := results2[len(results2)-1].(*types.Nil)
				if results2[len(results2)-1] != f.Pkg.TypesInfo.ObjectOf(id1) &&
					!isNil {
					continue
				}
				for i, v := range results1[:len(results1)-1] {
					if v != results2[i] {
						continue outer
					}
				}

				id1Obj := f.Pkg.TypesInfo.ObjectOf(id1)
				if id1Obj == nil {
					continue
				}
				_, idIface := id1Obj.Type().Underlying().(*types.Interface)
				_, retIface := f.Pkg.TypesInfo.TypeOf(ret.List[ret.NumFields()-1].Type).Underlying().(*types.Interface)

				if retIface && !idIface {
					// When the return value is an interface, but the
					// identifier is not, an explicit check for nil is
					// required to return an untyped nil.
					continue
				}

				f.Errorf(ifs, 1, "'if %s != nil { return %s }; return %s' can be simplified to 'return %s'",
					f.Render(expr.X), f.RenderArgs(ret1.Results),
					f.RenderArgs(ret2.Results), f.RenderArgs(ret1.Results))
			}
			return true
		}
		ast.Inspect(node, fn2)
		return true
	}
	f.Walk(fn1)
}

func LintReceiveIntoBlank(f *lint.File) {
	fn := func(node ast.Node) bool {
		stmt, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		if len(stmt.Lhs) != len(stmt.Rhs) {
			return true
		}
		for i, lh := range stmt.Lhs {
			rh := stmt.Rhs[i]
			if !lint.IsBlank(lh) {
				continue
			}
			expr, ok := rh.(*ast.UnaryExpr)
			if !ok {
				continue
			}
			if expr.Op != token.ARROW {
				continue
			}
			f.Errorf(lh, 1, "'_ = <-ch' can be simplified to '<-ch'")
		}
		return true
	}
	f.Walk(fn)
}

func LintFormatInt(f *lint.File) {
	checkBasic := func(v ast.Expr) bool {
		typ, ok := f.Pkg.TypesInfo.TypeOf(v).(*types.Basic)
		if !ok {
			return false
		}
		switch typ.Kind() {
		case types.Int, types.Int32:
			return true
		}
		return false
	}
	checkConst := func(v *ast.Ident) bool {
		c, ok := f.Pkg.TypesInfo.ObjectOf(v).(*types.Const)
		if !ok {
			return false
		}
		if c.Val().Kind() != constant.Int {
			return false
		}
		i, _ := constant.Int64Val(c.Val())
		return i <= math.MaxInt32
	}
	checkConstStrict := func(v *ast.Ident) bool {
		if !checkConst(v) {
			return false
		}
		basic, ok := f.Pkg.TypesInfo.ObjectOf(v).(*types.Const).Type().(*types.Basic)
		return ok && basic.Kind() == types.UntypedInt
	}

	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !lint.IsPkgDot(call.Fun, "strconv", "FormatInt") {
			return true
		}
		if len(call.Args) != 2 {
			return true
		}
		if lit, ok := call.Args[1].(*ast.BasicLit); !ok || lit.Value != "10" {
			return true
		}

		matches := false
		switch v := call.Args[0].(type) {
		case *ast.CallExpr:
			if len(v.Args) != 1 {
				return true
			}
			ident, ok := v.Fun.(*ast.Ident)
			if !ok {
				return true
			}
			obj, ok := f.Pkg.TypesInfo.ObjectOf(ident).(*types.TypeName)
			if !ok || obj.Parent() != types.Universe || obj.Name() != "int64" {
				return true
			}

			switch vv := v.Args[0].(type) {
			case *ast.BasicLit:
				i, _ := strconv.ParseInt(vv.Value, 10, 64)
				if i <= math.MaxInt32 {
					matches = true
				}
			case *ast.Ident:
				if checkConst(vv) || checkBasic(v.Args[0]) {
					matches = true
				}
			default:
				if checkBasic(v.Args[0]) {
					matches = true
				}
			}
		case *ast.BasicLit:
			if v.Kind != token.INT {
				return true
			}
			i, _ := strconv.ParseInt(v.Value, 10, 64)
			if i <= math.MaxInt32 {
				matches = true
			}
		case *ast.Ident:
			if checkConstStrict(v) {
				matches = true
			}
		}
		if matches {
			f.Errorf(call, 1, "should use strconv.Itoa instead of strconv.FormatInt")
		}
		return true
	}
	f.Walk(fn)
}

func LintSimplerStructConversion(f *lint.File) {
	fn := func(node ast.Node) bool {
		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		typ1 := f.Pkg.TypesInfo.TypeOf(lit.Type)
		if typ1 == nil {
			return true
		}
		// FIXME support pointer to struct
		s1, ok := typ1.Underlying().(*types.Struct)
		if !ok {
			return true
		}

		n := s1.NumFields()
		var typ2 types.Type
		var ident *ast.Ident
		getSelType := func(expr ast.Expr) (types.Type, *ast.Ident, bool) {
			sel, ok := expr.(*ast.SelectorExpr)
			if !ok {
				return nil, nil, false
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return nil, nil, false
			}
			typ := f.Pkg.TypesInfo.TypeOf(sel.X)
			return typ, ident, typ != nil
		}
		if len(lit.Elts) == 0 {
			return true
		}
		for i, elt := range lit.Elts {
			n--
			var t types.Type
			var id *ast.Ident
			var ok bool
			switch elt := elt.(type) {
			case *ast.SelectorExpr:
				t, id, ok = getSelType(elt)
				if !ok {
					return true
				}
				if i >= s1.NumFields() || s1.Field(i).Name() != elt.Sel.Name {
					return true
				}
			case *ast.KeyValueExpr:
				var sel *ast.SelectorExpr
				sel, ok = elt.Value.(*ast.SelectorExpr)
				if !ok {
					return true
				}

				if elt.Key.(*ast.Ident).Name != sel.Sel.Name {
					return true
				}
				t, id, ok = getSelType(elt.Value)
			}
			if !ok {
				return true
			}
			if typ2 != nil && typ2 != t {
				return true
			}
			if ident != nil && ident.Obj != id.Obj {
				return true
			}
			typ2 = t
			ident = id
		}

		if n != 0 {
			return true
		}

		if typ2 == nil {
			return true
		}

		s2, ok := typ2.Underlying().(*types.Struct)
		if !ok {
			return true
		}
		if typ1 == typ2 {
			return true
		}
		if !structsIdentical(s1, s2) {
			return true
		}
		f.Errorf(node, 1, "should use type conversion instead of struct literal")
		return true
	}
	f.Walk(fn)
}

func LintTrim(f *lint.File) {
	// TODO(dh): implement the same for suffix
	sameNonDynamic := func(node1, node2 ast.Node) bool {
		if reflect.TypeOf(node1) != reflect.TypeOf(node2) {
			return false
		}

		switch node1 := node1.(type) {
		case *ast.Ident:
			return node1.Obj == node2.(*ast.Ident).Obj
		case *ast.SelectorExpr:
			return f.Render(node1) == f.Render(node2)
		case *ast.IndexExpr:
			return f.Render(node1) == f.Render(node2)
		}
		return false
	}

	isLenOnIdent := func(fn ast.Expr, ident ast.Expr) bool {
		call, ok := fn.(*ast.CallExpr)
		if !ok {
			return false
		}
		if fn, ok := call.Fun.(*ast.Ident); !ok || fn.Name != "len" {
			return false
		}
		if len(call.Args) != 1 {
			return false
		}
		return sameNonDynamic(call.Args[0], ident)
	}

	fn := func(node ast.Node) bool {
		var pkg string
		var fun string

		ifstmt, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		if ifstmt.Init != nil {
			return true
		}
		if ifstmt.Else != nil {
			return true
		}
		if len(ifstmt.Body.List) != 1 {
			return true
		}
		condCall, ok := ifstmt.Cond.(*ast.CallExpr)
		if !ok {
			return true
		}
		call, ok := condCall.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if lint.IsIdent(call.X, "strings") {
			pkg = "strings"
		} else if lint.IsIdent(call.X, "bytes") {
			pkg = "bytes"
		} else {
			return true
		}
		if lint.IsIdent(call.Sel, "HasPrefix") {
			fun = "HasPrefix"
		} else if lint.IsIdent(call.Sel, "HasSuffix") {
			fun = "HasSuffix"
		} else {
			return true
		}

		assign, ok := ifstmt.Body.List[0].(*ast.AssignStmt)
		if !ok {
			return true
		}
		if assign.Tok != token.ASSIGN {
			return true
		}
		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}
		if !sameNonDynamic(condCall.Args[0], assign.Lhs[0]) {
			return true
		}
		slice, ok := assign.Rhs[0].(*ast.SliceExpr)
		if !ok {
			return true
		}
		if slice.Slice3 {
			return true
		}
		if !sameNonDynamic(slice.X, condCall.Args[0]) {
			return true
		}
		var index ast.Expr
		switch fun {
		case "HasPrefix":
			// TODO(dh) We could detect a High that is len(s), but another
			// rule will already flag that, anyway.
			if slice.High != nil {
				return true
			}
			index = slice.Low
		case "HasSuffix":
			if slice.Low != nil {
				n, ok := intLit(f, slice.Low)
				if !ok || n != 0 {
					return true
				}
			}
			index = slice.High
		}

		switch index := index.(type) {
		case *ast.CallExpr:
			if fun != "HasPrefix" {
				return true
			}
			if fn, ok := index.Fun.(*ast.Ident); !ok || fn.Name != "len" {
				return true
			}
			if len(index.Args) != 1 {
				return true
			}
			id3 := index.Args[0]
			switch oid3 := condCall.Args[1].(type) {
			case *ast.BasicLit:
				if pkg != "strings" {
					return false
				}
				lit, ok := id3.(*ast.BasicLit)
				if !ok {
					return true
				}
				s1, ok1 := stringLit(f, lit)
				s2, ok2 := stringLit(f, condCall.Args[1])
				if !ok1 || !ok2 || s1 != s2 {
					return true
				}
			default:
				if !sameNonDynamic(id3, oid3) {
					return true
				}
			}
		case *ast.BasicLit, *ast.Ident:
			if fun != "HasPrefix" {
				return true
			}
			if pkg != "strings" {
				return true
			}
			string, ok1 := stringLit(f, condCall.Args[1])
			int, ok2 := intLit(f, slice.Low)
			if !ok1 || !ok2 || int != int64(len(string)) {
				return true
			}
		case *ast.BinaryExpr:
			if fun != "HasSuffix" {
				return true
			}
			if index.Op != token.SUB {
				return true
			}
			if !isLenOnIdent(index.X, condCall.Args[0]) ||
				!isLenOnIdent(index.Y, condCall.Args[1]) {
				return true
			}
		default:
			return true
		}

		var replacement string
		switch fun {
		case "HasPrefix":
			replacement = "TrimPrefix"
		case "HasSuffix":
			replacement = "TrimSuffix"
		}
		f.Errorf(ifstmt, 1, "should replace this if statement with an unconditional %s.%s", pkg, replacement)
		return true
	}
	f.Walk(fn)
}

func stringLit(f *lint.File, expr ast.Expr) (string, bool) {
	tv := f.Pkg.TypesInfo.Types[expr]
	if tv.Value == nil {
		return "", false
	}
	if tv.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}

func intLit(f *lint.File, expr ast.Expr) (int64, bool) {
	tv := f.Pkg.TypesInfo.Types[expr]
	if tv.Value == nil {
		return 0, false
	}
	if tv.Value.Kind() != constant.Int {
		return 0, false
	}
	val, _ := constant.Int64Val(tv.Value)
	return val, true
}
