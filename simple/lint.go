// Package simple contains a linter for Go source code.
package simple // import "honnef.co/go/tools/simple"

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
	. "honnef.co/go/tools/arg"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/internal/sharedcheck"
	"honnef.co/go/tools/lint"
	. "honnef.co/go/tools/lint/lintdsl"
	"honnef.co/go/tools/lint/lintutil"
)

func LintSingleCaseSelect(pass *analysis.Pass) (interface{}, error) {
	q1 := lintutil.MustParse(`
		(ForStmt
			nil nil nil
			select@(SelectStmt
				(CommClause
					(Or
						(UnaryExpr "<-" _)
						(AssignStmt _ _ (UnaryExpr "<-" _)))
					_)))`)
	q2 := lintutil.MustParse(`(SelectStmt (CommClause _ _))`)

	seen := map[ast.Node]struct{}{}
	fn := func(node ast.Node) {
		if m, ok := Match(pass, q1, node); ok {
			seen[m.State["select"].(ast.Node)] = struct{}{}
			ReportNodefFG(pass, node, "should use for range instead of for { select {} }")
		} else if _, ok := Match(pass, q2, node); ok {
			if _, ok := seen[node]; !ok {
				ReportNodefFG(pass, node, "should use a simple channel send/receive instead of select with a single case")
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil), (*ast.SelectStmt)(nil)}, fn)
	return nil, nil
}

func LintLoopCopy(pass *analysis.Pass) (interface{}, error) {
	q1 := lintutil.MustParse(`
		(Or
			(RangeStmt
				key value ":=" src@(Ident _)
				[(AssignStmt
					(IndexExpr dst@(Ident _) key)
					"="
					value)])
			(RangeStmt
				key nil ":=" src@(Ident _)
				[(AssignStmt
					(IndexExpr dst@(Ident _) key)
					"="
					(IndexExpr src key))]))`)

	fn := func(node ast.Node) {
		loop := node.(*ast.RangeStmt)
		m, ok := Match(pass, q1, loop)
		if !ok {
			return
		}
		t1 := pass.TypesInfo.TypeOf(m.State["src"].(*ast.Ident))
		t2 := pass.TypesInfo.TypeOf(m.State["dst"].(*ast.Ident))
		if _, ok := t1.Underlying().(*types.Slice); !ok {
			return
		}
		if !types.Identical(t1, t2) {
			return
		}
		ReportNodefFG(pass, loop, "should use copy() instead of a loop")
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.RangeStmt)(nil)}, fn)
	return nil, nil
}

func LintIfBoolCmp(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		expr := node.(*ast.BinaryExpr)
		if expr.Op != token.EQL && expr.Op != token.NEQ {
			return
		}
		x := IsBoolConst(pass, expr.X)
		y := IsBoolConst(pass, expr.Y)
		if !x && !y {
			return
		}
		var other ast.Expr
		var val bool
		if x {
			val = BoolConst(pass, expr.X)
			other = expr.Y
		} else {
			val = BoolConst(pass, expr.Y)
			other = expr.X
		}
		basic, ok := pass.TypesInfo.TypeOf(other).Underlying().(*types.Basic)
		if !ok || basic.Kind() != types.Bool {
			return
		}
		op := ""
		if (expr.Op == token.EQL && !val) || (expr.Op == token.NEQ && val) {
			op = "!"
		}
		r := op + Render(pass, other)
		l1 := len(r)
		r = strings.TrimLeft(r, "!")
		if (l1-len(r))%2 == 1 {
			r = "!" + r
		}
		if IsInTest(pass, node) {
			return
		}
		ReportNodefFG(pass, expr, "should omit comparison to bool constant, can be simplified to %s", r)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func LintBytesBufferConversions(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`(CallExpr _ [(CallExpr sel@(SelectorExpr _ _) [])])`)
	fn := func(node ast.Node) {
		m, ok := Match(pass, q, node)
		if !ok {
			return
		}
		call := node.(*ast.CallExpr)
		sel := m.State["sel"].(*ast.SelectorExpr)

		typ := pass.TypesInfo.TypeOf(call.Fun)
		if typ == types.Universe.Lookup("string").Type() && IsCallToAST(pass, call.Args[0], "(*bytes.Buffer).Bytes") {
			ReportNodefFG(pass, call, "should use %v.String() instead of %v", Render(pass, sel.X), Render(pass, call))
		} else if typ, ok := typ.(*types.Slice); ok && typ.Elem() == types.Universe.Lookup("byte").Type() && IsCallToAST(pass, call.Args[0], "(*bytes.Buffer).String") {
			ReportNodefFG(pass, call, "should use %v.Bytes() instead of %v", Render(pass, sel.X), Render(pass, call))
		}

	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintStringsContains(pass *analysis.Pass) (interface{}, error) {
	// map of value to token to bool value
	allowed := map[int64]map[token.Token]bool{
		-1: {token.GTR: true, token.NEQ: true, token.EQL: false},
		0:  {token.GEQ: true, token.LSS: false},
	}
	fn := func(node ast.Node) {
		expr := node.(*ast.BinaryExpr)
		switch expr.Op {
		case token.GEQ, token.GTR, token.NEQ, token.LSS, token.EQL:
		default:
			return
		}

		value, ok := ExprToInt(pass, expr.Y)
		if !ok {
			return
		}

		allowedOps, ok := allowed[value]
		if !ok {
			return
		}
		b, ok := allowedOps[expr.Op]
		if !ok {
			return
		}

		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			return
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			return
		}
		funIdent := sel.Sel
		if pkgIdent.Name != "strings" && pkgIdent.Name != "bytes" {
			return
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
			return
		}

		prefix := ""
		if !b {
			prefix = "!"
		}
		ReportNodefFG(pass, node, "should use %s%s.%s(%s) instead", prefix, pkgIdent.Name, newFunc, RenderArgs(pass, call.Args))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func LintBytesCompare(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		expr := node.(*ast.BinaryExpr)
		if expr.Op != token.NEQ && expr.Op != token.EQL {
			return
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			return
		}
		if !IsCallToAST(pass, call, "bytes.Compare") {
			return
		}
		value, ok := ExprToInt(pass, expr.Y)
		if !ok || value != 0 {
			return
		}
		args := RenderArgs(pass, call.Args)
		prefix := ""
		if expr.Op == token.NEQ {
			prefix = "!"
		}
		ReportNodefFG(pass, node, "should use %sbytes.Equal(%s) instead", prefix, args)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func LintForTrue(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		loop := node.(*ast.ForStmt)
		if loop.Init != nil || loop.Post != nil {
			return
		}
		if !IsBoolConst(pass, loop.Cond) || !BoolConst(pass, loop.Cond) {
			return
		}
		ReportNodefFG(pass, loop, "should use for {} instead of for true {}")
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil)}, fn)
	return nil, nil
}

func LintRegexpRaw(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if !IsCallToAST(pass, call, "regexp.MustCompile") &&
			!IsCallToAST(pass, call, "regexp.Compile") {
			return
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		lit, ok := call.Args[Arg("regexp.Compile.expr")].(*ast.BasicLit)
		if !ok {
			// TODO(dominikh): support string concat, maybe support constants
			return
		}
		if lit.Kind != token.STRING {
			// invalid function call
			return
		}
		if lit.Value[0] != '"' {
			// already a raw string
			return
		}
		val := lit.Value
		if !strings.Contains(val, `\\`) {
			return
		}
		if strings.Contains(val, "`") {
			return
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
				return
			}
		}

		ReportNodefFG(pass, call, "should use raw string (`...`) with regexp.%s to avoid having to escape twice", sel.Sel.Name)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintIfReturn(pass *analysis.Pass) (interface{}, error) {
	qIf := lintutil.MustParse(`(IfStmt nil cond [(ReturnStmt [ret@(Ident _)])] nil)`)
	qRet := lintutil.MustParse(`(ReturnStmt [ret@(Ident _)])`)
	fn := func(node ast.Node) {
		block := node.(*ast.BlockStmt)
		l := len(block.List)
		if l < 2 {
			return
		}
		n1, n2 := block.List[l-2], block.List[l-1]

		if len(block.List) >= 3 {
			if _, ok := block.List[l-3].(*ast.IfStmt); ok {
				// Do not flag a series of if statements
				return
			}
		}
		m1, ok := Match(pass, qIf, n1)
		if !ok {
			return
		}
		m2, ok := Match(pass, qRet, n2)
		if !ok {
			return
		}

		if op, ok := m1.State["cond"].(*ast.BinaryExpr); ok {
			switch op.Op {
			case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
			default:
				return
			}
		}

		ret1 := m1.State["ret"].(*ast.Ident)
		if !IsBoolConst(pass, ret1) {
			return
		}
		ret2 := m2.State["ret"].(*ast.Ident)
		if !IsBoolConst(pass, ret2) {
			return
		}

		if ret1.Name == ret2.Name {
			// we want the function to return true and false, not the
			// same value both times.
			return
		}

		cond := m1.State["cond"].(ast.Expr)
		if ret1.Name == "false" {
			cond = negate(cond)
		}
		ReportNodefFG(pass, n1, "should use 'return %s' instead of 'if %s { return %s }; return %s'",
			Render(pass, cond),
			Render(pass, cond), Render(pass, ret1), Render(pass, ret2))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BlockStmt)(nil)}, fn)
	return nil, nil
}

func negate(expr ast.Expr) ast.Expr {
	switch expr := expr.(type) {
	case *ast.BinaryExpr:
		out := *expr
		switch expr.Op {
		case token.EQL:
			out.Op = token.NEQ
		case token.LSS:
			out.Op = token.GEQ
		case token.GTR:
			out.Op = token.LEQ
		case token.NEQ:
			out.Op = token.EQL
		case token.LEQ:
			out.Op = token.GTR
		case token.GEQ:
			out.Op = token.LEQ
		}
		return &out
	case *ast.Ident, *ast.CallExpr, *ast.IndexExpr:
		return &ast.UnaryExpr{
			Op: token.NOT,
			X:  expr,
		}
	default:
		return &ast.UnaryExpr{
			Op: token.NOT,
			X: &ast.ParenExpr{
				X: expr,
			},
		}
	}
}

// LintRedundantNilCheckWithLen checks for the following reduntant nil-checks:
//
//   if x == nil || len(x) == 0 {}
//   if x != nil && len(x) != 0 {}
//   if x != nil && len(x) == N {} (where N != 0)
//   if x != nil && len(x) > N {}
//   if x != nil && len(x) >= N {} (where N != 0)
//
func LintRedundantNilCheckWithLen(pass *analysis.Pass) (interface{}, error) {
	isConstZero := func(expr ast.Expr) (isConst bool, isZero bool) {
		_, ok := expr.(*ast.BasicLit)
		if ok {
			return true, IsZero(expr)
		}
		id, ok := expr.(*ast.Ident)
		if !ok {
			return false, false
		}
		c, ok := pass.TypesInfo.ObjectOf(id).(*types.Const)
		if !ok {
			return false, false
		}
		return true, c.Val().Kind() == constant.Int && c.Val().String() == "0"
	}

	fn := func(node ast.Node) {
		// check that expr is "x || y" or "x && y"
		expr := node.(*ast.BinaryExpr)
		if expr.Op != token.LOR && expr.Op != token.LAND {
			return
		}
		eqNil := expr.Op == token.LOR

		// check that x is "xx == nil" or "xx != nil"
		x, ok := expr.X.(*ast.BinaryExpr)
		if !ok {
			return
		}
		if eqNil && x.Op != token.EQL {
			return
		}
		if !eqNil && x.Op != token.NEQ {
			return
		}
		xx, ok := x.X.(*ast.Ident)
		if !ok {
			return
		}
		if !IsNil(pass, x.Y) {
			return
		}

		// check that y is "len(xx) == 0" or "len(xx) ... "
		y, ok := expr.Y.(*ast.BinaryExpr)
		if !ok {
			return
		}
		if eqNil && y.Op != token.EQL { // must be len(xx) *==* 0
			return
		}
		yx, ok := y.X.(*ast.CallExpr)
		if !ok {
			return
		}
		yxFun, ok := yx.Fun.(*ast.Ident)
		if !ok || yxFun.Name != "len" || len(yx.Args) != 1 {
			return
		}
		yxArg, ok := yx.Args[Arg("len.v")].(*ast.Ident)
		if !ok {
			return
		}
		if yxArg.Name != xx.Name {
			return
		}

		if eqNil && !IsZero(y.Y) { // must be len(x) == *0*
			return
		}

		if !eqNil {
			isConst, isZero := isConstZero(y.Y)
			if !isConst {
				return
			}
			switch y.Op {
			case token.EQL:
				// avoid false positive for "xx != nil && len(xx) == 0"
				if isZero {
					return
				}
			case token.GEQ:
				// avoid false positive for "xx != nil && len(xx) >= 0"
				if isZero {
					return
				}
			case token.NEQ:
				// avoid false positive for "xx != nil && len(xx) != <non-zero>"
				if !isZero {
					return
				}
			case token.GTR:
				// ok
			default:
				return
			}
		}

		// finally check that xx type is one of array, slice, map or chan
		// this is to prevent false positive in case if xx is a pointer to an array
		var nilType string
		switch pass.TypesInfo.TypeOf(xx).(type) {
		case *types.Slice:
			nilType = "nil slices"
		case *types.Map:
			nilType = "nil maps"
		case *types.Chan:
			nilType = "nil channels"
		default:
			return
		}
		ReportNodefFG(pass, expr, "should omit nil check; len() for %s is defined as zero", nilType)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func LintSlicing(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`(SliceExpr x@(Object _) _ (CallExpr (Builtin "len") [x]) nil)`)
	fn := func(node ast.Node) {
		if _, ok := Match(pass, q, node); ok {
			ReportNodefFG(pass, node, "should omit second index in slice, s[a:len(s)] is identical to s[a:]")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.SliceExpr)(nil)}, fn)
	return nil, nil
}

func refersTo(pass *analysis.Pass, expr ast.Expr, ident types.Object) bool {
	found := false
	fn := func(node ast.Node) bool {
		ident2, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if ident == pass.TypesInfo.ObjectOf(ident2) {
			found = true
			return false
		}
		return true
	}
	ast.Inspect(expr, fn)
	return found
}

func LintLoopAppend(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`
		(RangeStmt
			(Ident "_")
			val@(Object _)
			_
			x
			[(AssignStmt [lhs] "=" [(CallExpr (Builtin "append") [lhs val])])])
`)
	fn := func(node ast.Node) {
		m, ok := Match(pass, q, node)
		if !ok {
			return
		}

		val := m.State["val"].(types.Object)
		if refersTo(pass, m.State["lhs"].(ast.Expr), val) {
			return
		}

		src := pass.TypesInfo.TypeOf(m.State["x"].(ast.Expr))
		dst := pass.TypesInfo.TypeOf(m.State["lhs"].(ast.Expr))
		if !types.Identical(src, dst) {
			return
		}

		ReportNodefFG(pass, node, "should replace loop with %s = append(%s, %s...)",
			Render(pass, m.State["lhs"]), Render(pass, m.State["lhs"]), Render(pass, m.State["x"]))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.RangeStmt)(nil)}, fn)
	return nil, nil
}

func LintTimeSince(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`(CallExpr (SelectorExpr (CallExpr (Function "time.Now") []) (Function "(time.Time).Sub")) _)`)
	fn := func(node ast.Node) {
		if _, ok := Match(pass, q, node); ok {
			ReportNodefFG(pass, node, "should use time.Since instead of time.Now().Sub")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintTimeUntil(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`(CallExpr (Function "(time.Time).Sub") [(CallExpr (Function "time.Now") [])])`)
	if !IsGoVersion(pass, 8) {
		return nil, nil
	}
	fn := func(node ast.Node) {
		if _, ok := Match(pass, q, node); ok {
			ReportNodefFG(pass, node, "should use time.Until instead of t.Sub(time.Now())")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintUnnecessaryBlank(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`
		(AssignStmt
			[_ (Ident "_")]
			_
			(Or
				(IndexExpr _ _)
				(UnaryExpr "<-" _))) `)
	fn1 := func(node ast.Node) {
		if _, ok := Match(pass, q, node); !ok {
			return
		}
		cp := *node.(*ast.AssignStmt)
		cp.Lhs = cp.Lhs[0:1]
		ReportNodefFG(pass, node, "should write %s instead of %s", Render(pass, &cp), Render(pass, node))
	}

	fn2 := func(node ast.Node) {
		stmt := node.(*ast.AssignStmt)
		if len(stmt.Lhs) != len(stmt.Rhs) {
			return
		}
		for i, lh := range stmt.Lhs {
			rh := stmt.Rhs[i]
			if !IsBlank(lh) {
				continue
			}
			expr, ok := rh.(*ast.UnaryExpr)
			if !ok {
				continue
			}
			if expr.Op != token.ARROW {
				continue
			}
			ReportNodefFG(pass, lh, "'_ = <-ch' can be simplified to '<-ch'")
		}
	}

	fn3 := func(node ast.Node) {
		rs := node.(*ast.RangeStmt)

		// for x, _
		if !IsBlank(rs.Key) && IsBlank(rs.Value) {
			ReportNodefFG(pass, rs.Value, "should omit value from range; this loop is equivalent to `for %s %s range ...`", Render(pass, rs.Key), rs.Tok)
		}
		// for _, _ || for _
		if IsBlank(rs.Key) && (IsBlank(rs.Value) || rs.Value == nil) {
			ReportNodefFG(pass, rs.Key, "should omit values from range; this loop is equivalent to `for range ...`")
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.AssignStmt)(nil)}, fn1)
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.AssignStmt)(nil)}, fn2)
	if IsGoVersion(pass, 4) {
		pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.RangeStmt)(nil)}, fn3)
	}
	return nil, nil
}

func LintSimplerStructConversion(pass *analysis.Pass) (interface{}, error) {
	var skip ast.Node
	fn := func(node ast.Node) {
		// Do not suggest type conversion between pointers
		if unary, ok := node.(*ast.UnaryExpr); ok && unary.Op == token.AND {
			if lit, ok := unary.X.(*ast.CompositeLit); ok {
				skip = lit
			}
			return
		}

		if node == skip {
			return
		}

		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return
		}
		typ1, _ := pass.TypesInfo.TypeOf(lit.Type).(*types.Named)
		if typ1 == nil {
			return
		}
		s1, ok := typ1.Underlying().(*types.Struct)
		if !ok {
			return
		}

		var typ2 *types.Named
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
			typ := pass.TypesInfo.TypeOf(sel.X)
			return typ, ident, typ != nil
		}
		if len(lit.Elts) == 0 {
			return
		}
		if s1.NumFields() != len(lit.Elts) {
			return
		}
		for i, elt := range lit.Elts {
			var t types.Type
			var id *ast.Ident
			var ok bool
			switch elt := elt.(type) {
			case *ast.SelectorExpr:
				t, id, ok = getSelType(elt)
				if !ok {
					return
				}
				if i >= s1.NumFields() || s1.Field(i).Name() != elt.Sel.Name {
					return
				}
			case *ast.KeyValueExpr:
				var sel *ast.SelectorExpr
				sel, ok = elt.Value.(*ast.SelectorExpr)
				if !ok {
					return
				}

				if elt.Key.(*ast.Ident).Name != sel.Sel.Name {
					return
				}
				t, id, ok = getSelType(elt.Value)
			}
			if !ok {
				return
			}
			// All fields must be initialized from the same object
			if ident != nil && ident.Obj != id.Obj {
				return
			}
			typ2, _ = t.(*types.Named)
			if typ2 == nil {
				return
			}
			ident = id
		}

		if typ2 == nil {
			return
		}

		if typ1.Obj().Pkg() != typ2.Obj().Pkg() {
			// Do not suggest type conversions between different
			// packages. Types in different packages might only match
			// by coincidence. Furthermore, if the dependency ever
			// adds more fields to its type, it could break the code
			// that relies on the type conversion to work.
			return
		}

		s2, ok := typ2.Underlying().(*types.Struct)
		if !ok {
			return
		}
		if typ1 == typ2 {
			return
		}
		if IsGoVersion(pass, 8) {
			if !types.IdenticalIgnoreTags(s1, s2) {
				return
			}
		} else {
			if !types.Identical(s1, s2) {
				return
			}
		}
		ReportNodefFG(pass, node, "should convert %s (type %s) to %s instead of using struct literal",
			ident.Name, typ2.Obj().Name(), typ1.Obj().Name())
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.UnaryExpr)(nil), (*ast.CompositeLit)(nil)}, fn)
	return nil, nil
}

func LintTrim(pass *analysis.Pass) (interface{}, error) {
	sameNonDynamic := func(node1, node2 ast.Node) bool {
		if reflect.TypeOf(node1) != reflect.TypeOf(node2) {
			return false
		}

		switch node1 := node1.(type) {
		case *ast.Ident:
			return node1.Obj == node2.(*ast.Ident).Obj
		case *ast.SelectorExpr:
			return Render(pass, node1) == Render(pass, node2)
		case *ast.IndexExpr:
			return Render(pass, node1) == Render(pass, node2)
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
		return sameNonDynamic(call.Args[Arg("len.v")], ident)
	}

	fn := func(node ast.Node) {
		var pkg string
		var fun string

		ifstmt := node.(*ast.IfStmt)
		if ifstmt.Init != nil {
			return
		}
		if ifstmt.Else != nil {
			return
		}
		if len(ifstmt.Body.List) != 1 {
			return
		}
		condCall, ok := ifstmt.Cond.(*ast.CallExpr)
		if !ok {
			return
		}
		switch {
		case IsCallToAST(pass, condCall, "strings.HasPrefix"):
			pkg = "strings"
			fun = "HasPrefix"
		case IsCallToAST(pass, condCall, "strings.HasSuffix"):
			pkg = "strings"
			fun = "HasSuffix"
		case IsCallToAST(pass, condCall, "strings.Contains"):
			pkg = "strings"
			fun = "Contains"
		case IsCallToAST(pass, condCall, "bytes.HasPrefix"):
			pkg = "bytes"
			fun = "HasPrefix"
		case IsCallToAST(pass, condCall, "bytes.HasSuffix"):
			pkg = "bytes"
			fun = "HasSuffix"
		case IsCallToAST(pass, condCall, "bytes.Contains"):
			pkg = "bytes"
			fun = "Contains"
		default:
			return
		}

		assign, ok := ifstmt.Body.List[0].(*ast.AssignStmt)
		if !ok {
			return
		}
		if assign.Tok != token.ASSIGN {
			return
		}
		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return
		}
		if !sameNonDynamic(condCall.Args[0], assign.Lhs[0]) {
			return
		}

		switch rhs := assign.Rhs[0].(type) {
		case *ast.CallExpr:
			if len(rhs.Args) < 2 || !sameNonDynamic(condCall.Args[0], rhs.Args[0]) || !sameNonDynamic(condCall.Args[1], rhs.Args[1]) {
				return
			}
			if IsCallToAST(pass, condCall, "strings.HasPrefix") && IsCallToAST(pass, rhs, "strings.TrimPrefix") ||
				IsCallToAST(pass, condCall, "strings.HasSuffix") && IsCallToAST(pass, rhs, "strings.TrimSuffix") ||
				IsCallToAST(pass, condCall, "strings.Contains") && IsCallToAST(pass, rhs, "strings.Replace") ||
				IsCallToAST(pass, condCall, "bytes.HasPrefix") && IsCallToAST(pass, rhs, "bytes.TrimPrefix") ||
				IsCallToAST(pass, condCall, "bytes.HasSuffix") && IsCallToAST(pass, rhs, "bytes.TrimSuffix") ||
				IsCallToAST(pass, condCall, "bytes.Contains") && IsCallToAST(pass, rhs, "bytes.Replace") {
				ReportNodefFG(pass, ifstmt, "should replace this if statement with an unconditional %s", CallNameAST(pass, rhs))
			}
			return
		case *ast.SliceExpr:
			slice := rhs
			if !ok {
				return
			}
			if slice.Slice3 {
				return
			}
			if !sameNonDynamic(slice.X, condCall.Args[0]) {
				return
			}
			var index ast.Expr
			switch fun {
			case "HasPrefix":
				// TODO(dh) We could detect a High that is len(s), but another
				// rule will already flag that, anyway.
				if slice.High != nil {
					return
				}
				index = slice.Low
			case "HasSuffix":
				if slice.Low != nil {
					n, ok := ExprToInt(pass, slice.Low)
					if !ok || n != 0 {
						return
					}
				}
				index = slice.High
			}

			switch index := index.(type) {
			case *ast.CallExpr:
				if fun != "HasPrefix" {
					return
				}
				if fn, ok := index.Fun.(*ast.Ident); !ok || fn.Name != "len" {
					return
				}
				if len(index.Args) != 1 {
					return
				}
				id3 := index.Args[Arg("len.v")]
				switch oid3 := condCall.Args[1].(type) {
				case *ast.BasicLit:
					if pkg != "strings" {
						return
					}
					lit, ok := id3.(*ast.BasicLit)
					if !ok {
						return
					}
					s1, ok1 := ExprToString(pass, lit)
					s2, ok2 := ExprToString(pass, condCall.Args[1])
					if !ok1 || !ok2 || s1 != s2 {
						return
					}
				default:
					if !sameNonDynamic(id3, oid3) {
						return
					}
				}
			case *ast.BasicLit, *ast.Ident:
				if fun != "HasPrefix" {
					return
				}
				if pkg != "strings" {
					return
				}
				string, ok1 := ExprToString(pass, condCall.Args[1])
				int, ok2 := ExprToInt(pass, slice.Low)
				if !ok1 || !ok2 || int != int64(len(string)) {
					return
				}
			case *ast.BinaryExpr:
				if fun != "HasSuffix" {
					return
				}
				if index.Op != token.SUB {
					return
				}
				if !isLenOnIdent(index.X, condCall.Args[0]) ||
					!isLenOnIdent(index.Y, condCall.Args[1]) {
					return
				}
			default:
				return
			}

			var replacement string
			switch fun {
			case "HasPrefix":
				replacement = "TrimPrefix"
			case "HasSuffix":
				replacement = "TrimSuffix"
			}
			ReportNodefFG(pass, ifstmt, "should replace this if statement with an unconditional %s.%s", pkg, replacement)
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

func LintLoopSlide(pass *analysis.Pass) (interface{}, error) {
	// TODO(dh): detect bs[i+offset] in addition to bs[offset+i]
	// TODO(dh): consider merging this function with LintLoopCopy
	// TODO(dh): detect length that is an expression, not a variable name
	// TODO(dh): support sliding to a different offset than the beginning of the slice

	q := lintutil.MustParse(`
		(ForStmt
			(AssignStmt initvar@(Ident _) _ (BasicLit "INT" "0"))
			(BinaryExpr initvar "<" limit@(Ident _))
			(IncDecStmt initvar "++")
			[(AssignStmt
				(IndexExpr slice@(Ident _) initvar)
				"="
				(IndexExpr slice (BinaryExpr offset@(Ident _) "+" initvar)))])
		`)

	fn := func(node ast.Node) {
		/*
			for i := 0; i < n; i++ {
				bs[i] = bs[offset+i]
			}

						↓

			copy(bs[:n], bs[offset:offset+n])
		*/

		loop := node.(*ast.ForStmt)
		m, ok := Match(pass, q, loop)
		if !ok {
			return
		}
		if _, ok := pass.TypesInfo.TypeOf(m.State["slice"].(*ast.Ident)).Underlying().(*types.Slice); !ok {
			return
		}

		ReportNodefFG(pass, loop, "should use copy(%s[:%s], %s[%s:]) instead", Render(pass, m.State["slice"]), Render(pass, m.State["limit"]), Render(pass, m.State["slice"]), Render(pass, m.State["offset"]))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil)}, fn)
	return nil, nil
}

func LintMakeLenCap(pass *analysis.Pass) (interface{}, error) {
	q1 := lintutil.MustParse(`(CallExpr (Builtin "make") [typ size@(BasicLit "INT" "0")])`)
	q2 := lintutil.MustParse(`(CallExpr (Builtin "make") [typ size size])`)
	fn := func(node ast.Node) {
		if m, ok := Match(pass, q1, node); ok {
			T := m.State["typ"].(ast.Expr)
			size := m.State["size"].(ast.Node)
			if _, ok := pass.TypesInfo.TypeOf(T).Underlying().(*types.Slice); ok {
				return
			}
			ReportNodefFG(pass, size, "should use make(%s) instead", Render(pass, T))
		} else if m, ok := Match(pass, q2, node); ok {
			// TODO(dh): don't consider sizes identical if they're
			// dynamic. for example: make(T, <-ch, <-ch).
			T := m.State["typ"].(ast.Expr)
			size := m.State["size"].(ast.Node)
			ReportNodefFG(pass, size,
				"should use make(%s, %s) instead", Render(pass, T), Render(pass, size))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintAssertNotNil(pass *analysis.Pass) (interface{}, error) {
	fn1Q := lintutil.MustParse(`
		(IfStmt
			(AssignStmt [(Ident "_") ok@(Object _)] _ [(TypeAssertExpr assert@(Object _) _)])
			(Or
				(BinaryExpr ok "&&" (BinaryExpr assert "!=" (Builtin "nil")))
				(BinaryExpr (BinaryExpr assert "!=" (Builtin "nil")) "&&" ok))
			_
			_)`)
	fn2Q := lintutil.MustParse(`
		(IfStmt
			nil
			(BinaryExpr lhs@(Object _) "!=" (Builtin "nil"))
			[
				ifstmt@(IfStmt
					(AssignStmt [(Ident "_") ok@(Object _)] _ [(TypeAssertExpr lhs _)])
					ok
					_
					_)
			]
			nil)`)
	fn1 := func(node ast.Node) {
		m, ok := Match(pass, fn1Q, node)
		if !ok {
			return
		}
		assert := m.State["assert"].(types.Object)
		assign := m.State["ok"].(types.Object)
		ReportNodefFG(pass, node, "when %s is true, %s can't be nil", assign.Name(), assert.Name())
	}
	fn2 := func(node ast.Node) {
		m, ok := Match(pass, fn2Q, node)
		if !ok {
			return
		}
		ifstmt := m.State["ifstmt"].(*ast.IfStmt)
		lhs := m.State["lhs"].(types.Object)
		assignIdent := m.State["ok"].(types.Object)
		ReportNodefFG(pass, ifstmt, "when %s is true, %s can't be nil", assignIdent.Name(), lhs.Name())
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn1)
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn2)
	return nil, nil
}

func LintDeclareAssign(pass *analysis.Pass) (interface{}, error) {
	hasMultipleAssignments := func(root ast.Node, ident *ast.Ident) bool {
		num := 0
		ast.Inspect(root, func(node ast.Node) bool {
			if num >= 2 {
				return false
			}
			assign, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for _, lhs := range assign.Lhs {
				if oident, ok := lhs.(*ast.Ident); ok {
					if oident.Obj == ident.Obj {
						num++
					}
				}
			}

			return true
		})
		return num >= 2
	}
	fn := func(node ast.Node) {
		block := node.(*ast.BlockStmt)
		if len(block.List) < 2 {
			return
		}
		for i, stmt := range block.List[:len(block.List)-1] {
			_ = i
			decl, ok := stmt.(*ast.DeclStmt)
			if !ok {
				continue
			}
			gdecl, ok := decl.Decl.(*ast.GenDecl)
			if !ok || gdecl.Tok != token.VAR || len(gdecl.Specs) != 1 {
				continue
			}
			vspec, ok := gdecl.Specs[0].(*ast.ValueSpec)
			if !ok || len(vspec.Names) != 1 || len(vspec.Values) != 0 {
				continue
			}

			assign, ok := block.List[i+1].(*ast.AssignStmt)
			if !ok || assign.Tok != token.ASSIGN {
				continue
			}
			if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
				continue
			}
			ident, ok := assign.Lhs[0].(*ast.Ident)
			if !ok {
				continue
			}
			if vspec.Names[0].Obj != ident.Obj {
				continue
			}

			if refersTo(pass, assign.Rhs[0], pass.TypesInfo.ObjectOf(ident)) {
				continue
			}
			if hasMultipleAssignments(block, ident) {
				continue
			}

			ReportNodefFG(pass, decl, "should merge variable declaration with assignment on next line")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BlockStmt)(nil)}, fn)
	return nil, nil
}

func LintRedundantBreak(pass *analysis.Pass) (interface{}, error) {
	fn1 := func(node ast.Node) {
		clause := node.(*ast.CaseClause)
		if len(clause.Body) < 2 {
			return
		}
		branch, ok := clause.Body[len(clause.Body)-1].(*ast.BranchStmt)
		if !ok || branch.Tok != token.BREAK || branch.Label != nil {
			return
		}
		ReportNodefFG(pass, branch, "redundant break statement")
	}
	fn2 := func(node ast.Node) {
		var ret *ast.FieldList
		var body *ast.BlockStmt
		switch x := node.(type) {
		case *ast.FuncDecl:
			ret = x.Type.Results
			body = x.Body
		case *ast.FuncLit:
			ret = x.Type.Results
			body = x.Body
		default:
			ExhaustiveTypeSwitch(node)
		}
		// if the func has results, a return can't be redundant.
		// similarly, if there are no statements, there can be
		// no return.
		if ret != nil || body == nil || len(body.List) < 1 {
			return
		}
		rst, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
		if !ok {
			return
		}
		// we don't need to check rst.Results as we already
		// checked x.Type.Results to be nil.
		ReportNodefFG(pass, rst, "redundant return statement")
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CaseClause)(nil)}, fn1)
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)}, fn2)
	return nil, nil
}

func isStringer(T types.Type, msCache *typeutil.MethodSetCache) bool {
	ms := msCache.MethodSet(T)
	sel := ms.Lookup(nil, "String")
	if sel == nil {
		return false
	}
	fn, ok := sel.Obj().(*types.Func)
	if !ok {
		// should be unreachable
		return false
	}
	sig := fn.Type().(*types.Signature)
	if sig.Params().Len() != 0 {
		return false
	}
	if sig.Results().Len() != 1 {
		return false
	}
	if !IsType(sig.Results().At(0).Type(), "string") {
		return false
	}
	return true
}

func LintRedundantSprintf(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`(CallExpr (Function "fmt.Sprintf") [format arg])`)
	fn := func(node ast.Node) {
		m, ok := Match(pass, q, node)
		if !ok {
			return
		}

		format := m.State["format"].(ast.Expr)
		arg := m.State["arg"].(ast.Expr)
		if s, ok := ExprToString(pass, format); !ok || s != "%s" {
			return
		}
		typ := pass.TypesInfo.TypeOf(arg)

		ssapkg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).Pkg
		if types.TypeString(typ, nil) != "reflect.Value" && isStringer(typ, &ssapkg.Prog.MethodSets) {
			ReportNodef(pass, node, "should use String() instead of fmt.Sprintf")
			return
		}

		if typ.Underlying() == types.Universe.Lookup("string").Type() {
			if typ == types.Universe.Lookup("string").Type() {
				ReportNodefFG(pass, node, "the argument is already a string, there's no need to use fmt.Sprintf")
			} else {
				ReportNodefFG(pass, node, "the argument's underlying type is a string, should use a simple conversion instead of fmt.Sprintf")
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintErrorsNewSprintf(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`(CallExpr (Function "errors.New") [(CallExpr (Function "fmt.Sprintf") _)])`)
	fn := func(node ast.Node) {
		if _, ok := Match(pass, q, node); ok {
			ReportNodefFG(pass, node, "should use fmt.Errorf(...) instead of errors.New(fmt.Sprintf(...))")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintRangeStringRunes(pass *analysis.Pass) (interface{}, error) {
	return sharedcheck.CheckRangeStringRunes(pass)
}

func LintNilCheckAroundRange(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`
		(IfStmt
			nil
			(BinaryExpr x@(Object _) "!=" (Builtin "nil"))
			[(RangeStmt _ _ _ x _)]
			nil)`)
	fn := func(node ast.Node) {
		m, ok := Match(pass, q, node)
		if !ok {
			return
		}
		switch m.State["x"].(types.Object).Type().Underlying().(type) {
		case *types.Slice, *types.Map:
			ReportNodefFG(pass, node, "unnecessary nil check around range")

		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

func isPermissibleSort(pass *analysis.Pass, node ast.Node) bool {
	call := node.(*ast.CallExpr)
	typeconv, ok := call.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	sel, ok := typeconv.Fun.(*ast.SelectorExpr)
	if !ok {
		return true
	}
	name := SelectorName(pass, sel)
	switch name {
	case "sort.IntSlice", "sort.Float64Slice", "sort.StringSlice":
	default:
		return true
	}

	return false
}

func LintSortHelpers(pass *analysis.Pass) (interface{}, error) {
	type Error struct {
		node ast.Node
		msg  string
	}
	var allErrors []Error
	fn := func(node ast.Node) {
		var body *ast.BlockStmt
		switch node := node.(type) {
		case *ast.FuncLit:
			body = node.Body
		case *ast.FuncDecl:
			body = node.Body
		default:
			ExhaustiveTypeSwitch(node)
		}
		if body == nil {
			return
		}

		var errors []Error
		permissible := false
		fnSorts := func(node ast.Node) bool {
			if permissible {
				return false
			}
			if !IsCallToAST(pass, node, "sort.Sort") {
				return true
			}
			if isPermissibleSort(pass, node) {
				permissible = true
				return false
			}
			call := node.(*ast.CallExpr)
			typeconv := call.Args[Arg("sort.Sort.data")].(*ast.CallExpr)
			sel := typeconv.Fun.(*ast.SelectorExpr)
			name := SelectorName(pass, sel)

			switch name {
			case "sort.IntSlice":
				errors = append(errors, Error{node, "should use sort.Ints(...) instead of sort.Sort(sort.IntSlice(...))"})
			case "sort.Float64Slice":
				errors = append(errors, Error{node, "should use sort.Float64s(...) instead of sort.Sort(sort.Float64Slice(...))"})
			case "sort.StringSlice":
				errors = append(errors, Error{node, "should use sort.Strings(...) instead of sort.Sort(sort.StringSlice(...))"})
			}
			return true
		}
		ast.Inspect(body, fnSorts)

		if permissible {
			return
		}
		allErrors = append(allErrors, errors...)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.FuncLit)(nil), (*ast.FuncDecl)(nil)}, fn)
	sort.Slice(allErrors, func(i, j int) bool {
		return allErrors[i].node.Pos() < allErrors[j].node.Pos()
	})
	var prev token.Pos
	for _, err := range allErrors {
		if err.node.Pos() == prev {
			continue
		}
		prev = err.node.Pos()
		ReportNodefFG(pass, err.node, "%s", err.msg)
	}
	return nil, nil
}

func LintGuardedDelete(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`
		(IfStmt
			(AssignStmt
				[(Ident "_") ok@(Ident _)]
				":="
				(IndexExpr m key))
			ok
			[(CallExpr (Builtin "delete") [m key])]
			nil)`)

	fn := func(node ast.Node) {
		if _, ok := Match(pass, q, node); !ok {
			return
		}
		ReportNodefFG(pass, node, "unnecessary guard around call to delete")
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

func LintSimplifyTypeSwitch(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`
		(TypeSwitchStmt
			nil
			expr@(TypeAssertExpr ident@(Ident _) _)
			body)`)
	fn := func(node ast.Node) {
		m, ok := Match(pass, q, node)
		if !ok {
			return
		}
		stmt := node.(*ast.TypeSwitchStmt)
		expr := m.State["expr"].(ast.Expr)
		ident := m.State["ident"].(*ast.Ident)

		x := pass.TypesInfo.ObjectOf(ident)
		var allOffenders []ast.Node
		for _, clause := range stmt.Body.List {
			clause := clause.(*ast.CaseClause)
			if len(clause.List) != 1 {
				continue
			}
			hasUnrelatedAssertion := false
			var offenders []ast.Node
			ast.Inspect(clause, func(node ast.Node) bool {
				assert2, ok := node.(*ast.TypeAssertExpr)
				if !ok {
					return true
				}
				ident, ok := assert2.X.(*ast.Ident)
				if !ok {
					hasUnrelatedAssertion = true
					return false
				}
				if pass.TypesInfo.ObjectOf(ident) != x {
					hasUnrelatedAssertion = true
					return false
				}

				if !types.Identical(pass.TypesInfo.TypeOf(clause.List[0]), pass.TypesInfo.TypeOf(assert2.Type)) {
					hasUnrelatedAssertion = true
					return false
				}
				offenders = append(offenders, assert2)
				return true
			})
			if !hasUnrelatedAssertion {
				// don't flag cases that have other type assertions
				// unrelated to the one in the case clause. often
				// times, this is done for symmetry, when two
				// different values have to be asserted to the same
				// type.
				allOffenders = append(allOffenders, offenders...)
			}
		}
		if len(allOffenders) != 0 {
			at := ""
			for _, offender := range allOffenders {
				pos := lint.DisplayPosition(pass.Fset, offender.Pos())
				at += "\n\t" + pos.String()
			}
			ReportNodefFG(pass, expr, "assigning the result of this type assertion to a variable (switch %s := %s.(type)) could eliminate the following type assertions:%s", Render(pass, ident), Render(pass, ident), at)
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.TypeSwitchStmt)(nil)}, fn)
	return nil, nil
}

func LintRedundantCanonicalHeaderKey(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if !IsCallToAST(pass, call, "(net/http.Header).Add") &&
			!IsCallToAST(pass, call, "(net/http.Header).Del") &&
			!IsCallToAST(pass, call, "(net/http.Header).Get") &&
			!IsCallToAST(pass, call, "(net/http.Header).Set") {
			return
		}

		if !IsCallToAST(pass, call.Args[0], "net/http.CanonicalHeaderKey") {
			return
		}

		ReportNodefFG(pass, call, "calling net/http.CanonicalHeaderKey on the 'key' argument of %s is redundant", CallNameAST(pass, call))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func LintUnnecessaryGuard(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`
		(Or
			(IfStmt
				(AssignStmt [(Ident "_") ok@(Ident _)] ":=" indexexpr@(IndexExpr _ _))
				ok
				(AssignStmt indexexpr "=" (CallExpr (Builtin "append") indexexpr:values))
				(AssignStmt indexexpr "=" (CompositeLit _ values)))
			(IfStmt
				(AssignStmt [(Ident "_") ok] ":=" indexexpr@(IndexExpr _ _))
				ok
				(AssignStmt indexexpr "+=" value)
				(AssignStmt indexexpr "=" value))
			(IfStmt
				(AssignStmt [(Ident "_") ok] ":=" indexexpr@(IndexExpr _ _))
				ok
				(IncDecStmt indexexpr "++")
				(AssignStmt indexexpr "=" (BasicLit "INT" "1"))))`)
	fn := func(node ast.Node) {
		if m, ok := Match(pass, q, node); ok {
			if lintutil.MayHaveSideEffects(m.State["indexexpr"].(ast.Expr)) {
				return
			}
			ReportNodef(pass, node, "unnecessary guard around map access")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

func LintElaborateSleep(pass *analysis.Pass) (interface{}, error) {
	q := lintutil.MustParse(`(SelectStmt (CommClause (UnaryExpr "<-" (CallExpr (Function "time.After") [_])) _))`)
	fn := func(node ast.Node) {
		if _, ok := Match(pass, q, node); ok {
			ReportNodefFG(pass, node, "should use time.Sleep instead of elaborate way of sleeping")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.SelectStmt)(nil)}, fn)
	return nil, nil
}
