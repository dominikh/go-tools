// Package simple contains a linter for Go source code.
package simple // import "honnef.co/go/tools/simple"

import (
	"fmt"
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
	"honnef.co/go/tools/code"
	"honnef.co/go/tools/edit"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/internal/sharedcheck"
	"honnef.co/go/tools/lint"
	. "honnef.co/go/tools/lint/lintdsl"
	"honnef.co/go/tools/pattern"
	"honnef.co/go/tools/report"
)

var (
	checkSingleCaseSelectQ1 = pattern.MustParse(`
		(ForStmt
			nil nil nil
			select@(SelectStmt
				(CommClause
					(Or
						(UnaryExpr "<-" _)
						(AssignStmt _ _ (UnaryExpr "<-" _)))
					_)))`)
	checkSingleCaseSelectQ2 = pattern.MustParse(`(SelectStmt (CommClause _ _))`)
)

func CheckSingleCaseSelect(pass *analysis.Pass) (interface{}, error) {
	seen := map[ast.Node]struct{}{}
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkSingleCaseSelectQ1, node); ok {
			seen[m.State["select"].(ast.Node)] = struct{}{}
			report.NodefFG(pass, node, "should use for range instead of for { select {} }")
		} else if _, ok := Match(pass, checkSingleCaseSelectQ2, node); ok {
			if _, ok := seen[node]; !ok {
				report.NodefFG(pass, node, "should use a simple channel send/receive instead of select with a single case")
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil), (*ast.SelectStmt)(nil)}, fn)
	return nil, nil
}

var (
	checkLoopCopyQ = pattern.MustParse(`
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
	checkLoopCopyR = pattern.MustParse(`(CallExpr (Ident "copy") [dst src])`)
)

func CheckLoopCopy(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, edits, ok := MatchAndEdit(pass, checkLoopCopyQ, checkLoopCopyR, node)
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

		tv, err := types.Eval(pass.Fset, pass.Pkg, node.Pos(), "copy")
		if err == nil && tv.IsBuiltin() {
			report.NodeFG(pass, node,
				"should use copy() instead of a loop",
				edit.Fix("replace loop with call to copy()", edits...))
		} else {
			report.NodeFG(pass, node, "should use copy() instead of a loop")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.RangeStmt)(nil)}, fn)
	return nil, nil
}

func CheckIfBoolCmp(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if code.IsInTest(pass, node) {
			return
		}

		expr := node.(*ast.BinaryExpr)
		if expr.Op != token.EQL && expr.Op != token.NEQ {
			return
		}
		x := code.IsBoolConst(pass, expr.X)
		y := code.IsBoolConst(pass, expr.Y)
		if !x && !y {
			return
		}
		var other ast.Expr
		var val bool
		if x {
			val = code.BoolConst(pass, expr.X)
			other = expr.Y
		} else {
			val = code.BoolConst(pass, expr.Y)
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
		r := op + report.Render(pass, other)
		l1 := len(r)
		r = strings.TrimLeft(r, "!")
		if (l1-len(r))%2 == 1 {
			r = "!" + r
		}
		report.NodeFG(pass, expr, fmt.Sprintf("should omit comparison to bool constant, can be simplified to %s", r),
			edit.Fix("simplify bool comparison", edit.ReplaceWithString(pass.Fset, expr, r)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkBytesBufferConversionsQ  = pattern.MustParse(`(CallExpr _ [(CallExpr sel@(SelectorExpr recv _) [])])`)
	checkBytesBufferConversionsRs = pattern.MustParse(`(CallExpr (SelectorExpr recv (Ident "String")) [])`)
	checkBytesBufferConversionsRb = pattern.MustParse(`(CallExpr (SelectorExpr recv (Ident "Bytes")) [])`)
)

func CheckBytesBufferConversions(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkBytesBufferConversionsQ, node)
		if !ok {
			return
		}
		call := node.(*ast.CallExpr)
		sel := m.State["sel"].(*ast.SelectorExpr)

		typ := pass.TypesInfo.TypeOf(call.Fun)
		if typ == types.Universe.Lookup("string").Type() && code.IsCallToAST(pass, call.Args[0], "(*bytes.Buffer).Bytes") {
			report.NodeFG(pass, call, fmt.Sprintf("should use %v.String() instead of %v", report.Render(pass, sel.X), report.Render(pass, call)),
				edit.Fix("simplify conversion", edit.ReplaceWithPattern(pass, checkBytesBufferConversionsRs, m.State, node)))
		} else if typ, ok := typ.(*types.Slice); ok && typ.Elem() == types.Universe.Lookup("byte").Type() && code.IsCallToAST(pass, call.Args[0], "(*bytes.Buffer).String") {
			report.NodeFG(pass, call, fmt.Sprintf("should use %v.Bytes() instead of %v", report.Render(pass, sel.X), report.Render(pass, call)),
				edit.Fix("simplify conversion", edit.ReplaceWithPattern(pass, checkBytesBufferConversionsRb, m.State, node)))
		}

	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckStringsContains(pass *analysis.Pass) (interface{}, error) {
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

		value, ok := code.ExprToInt(pass, expr.Y)
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

		var r ast.Expr
		switch funIdent.Name {
		case "IndexRune":
			r = &ast.SelectorExpr{
				X:   pkgIdent,
				Sel: &ast.Ident{Name: "ContainsRune"},
			}
		case "IndexAny":
			r = &ast.SelectorExpr{
				X:   pkgIdent,
				Sel: &ast.Ident{Name: "ContainsAny"},
			}
		case "Index":
			r = &ast.SelectorExpr{
				X:   pkgIdent,
				Sel: &ast.Ident{Name: "Contains"},
			}
		default:
			return
		}

		r = &ast.CallExpr{
			Fun:  r,
			Args: call.Args,
		}
		if !b {
			r = &ast.UnaryExpr{
				Op: token.NOT,
				X:  r,
			}
		}

		report.NodeFG(pass, node, fmt.Sprintf("should use %s instead", report.Render(pass, r)),
			edit.Fix(fmt.Sprintf("simplify use of %s", report.Render(pass, call.Fun)), edit.ReplaceWithNode(pass.Fset, node, r)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkBytesCompareQ  = pattern.MustParse(`(BinaryExpr (CallExpr (Function "bytes.Compare") args) op@(Or "==" "!=") (BasicLit "INT" "0"))`)
	checkBytesCompareRn = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "bytes") (Ident "Equal")) args)`)
	checkBytesCompareRe = pattern.MustParse(`(UnaryExpr "!" (CallExpr (SelectorExpr (Ident "bytes") (Ident "Equal")) args))`)
)

func CheckBytesCompare(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkBytesCompareQ, node)
		if !ok {
			return
		}

		args := report.RenderArgs(pass, m.State["args"].([]ast.Expr))
		prefix := ""
		if m.State["op"].(token.Token) == token.NEQ {
			prefix = "!"
		}

		var fix analysis.SuggestedFix
		switch tok := m.State["op"].(token.Token); tok {
		case token.EQL:
			fix = edit.Fix("simplify use of bytes.Compare", edit.ReplaceWithPattern(pass, checkBytesCompareRe, m.State, node))
		case token.NEQ:
			fix = edit.Fix("simplify use of bytes.Compare", edit.ReplaceWithPattern(pass, checkBytesCompareRn, m.State, node))
		default:
			panic(fmt.Sprintf("unexpected token %v", tok))
		}
		report.NodeFG(pass, node, fmt.Sprintf("should use %sbytes.Equal(%s) instead", prefix, args), fix)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func CheckForTrue(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		loop := node.(*ast.ForStmt)
		if loop.Init != nil || loop.Post != nil {
			return
		}
		if !code.IsBoolConst(pass, loop.Cond) || !code.BoolConst(pass, loop.Cond) {
			return
		}
		report.NodefFG(pass, loop, "should use for {} instead of for true {}")
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil)}, fn)
	return nil, nil
}

func CheckRegexpRaw(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if !code.IsCallToAnyAST(pass, call, "regexp.MustCompile", "regexp.Compile") {
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

		report.NodefFG(pass, call, "should use raw string (`...`) with regexp.%s to avoid having to escape twice", sel.Sel.Name)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkIfReturnQIf  = pattern.MustParse(`(IfStmt nil cond [(ReturnStmt [ret@(Ident _)])] nil)`)
	checkIfReturnQRet = pattern.MustParse(`(ReturnStmt [ret@(Ident _)])`)
)

func CheckIfReturn(pass *analysis.Pass) (interface{}, error) {
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
		m1, ok := Match(pass, checkIfReturnQIf, n1)
		if !ok {
			return
		}
		m2, ok := Match(pass, checkIfReturnQRet, n2)
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
		if !code.IsBoolConst(pass, ret1) {
			return
		}
		ret2 := m2.State["ret"].(*ast.Ident)
		if !code.IsBoolConst(pass, ret2) {
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
		report.NodefFG(pass, n1, "should use 'return %s' instead of 'if %s { return %s }; return %s'",
			report.Render(pass, cond),
			report.Render(pass, cond), report.Render(pass, ret1), report.Render(pass, ret2))
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

// CheckRedundantNilCheckWithLen checks for the following redundant nil-checks:
//
//   if x == nil || len(x) == 0 {}
//   if x != nil && len(x) != 0 {}
//   if x != nil && len(x) == N {} (where N != 0)
//   if x != nil && len(x) > N {}
//   if x != nil && len(x) >= N {} (where N != 0)
//
func CheckRedundantNilCheckWithLen(pass *analysis.Pass) (interface{}, error) {
	isConstZero := func(expr ast.Expr) (isConst bool, isZero bool) {
		_, ok := expr.(*ast.BasicLit)
		if ok {
			return true, code.IsIntLiteral(expr, "0")
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
		if !code.IsNil(pass, x.Y) {
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

		if eqNil && !code.IsIntLiteral(y.Y, "0") { // must be len(x) == *0*
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
		report.NodefFG(pass, expr, "should omit nil check; len() for %s is defined as zero", nilType)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

var checkSlicingQ = pattern.MustParse(`(SliceExpr x@(Object _) low (CallExpr (Builtin "len") [x]) nil)`)

func CheckSlicing(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if _, ok := Match(pass, checkSlicingQ, node); ok {
			expr := node.(*ast.SliceExpr)
			report.NodeFG(pass, expr.High,
				"should omit second index in slice, s[a:len(s)] is identical to s[a:]",
				edit.Fix("simplify slice expression", edit.Delete(expr.High)))
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

var checkLoopAppendQ = pattern.MustParse(`
	(RangeStmt
		(Ident "_")
		val@(Object _)
		_
		x
		[(AssignStmt [lhs] "=" [(CallExpr (Builtin "append") [lhs val])])]) `)

func CheckLoopAppend(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkLoopAppendQ, node)
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

		r := &ast.AssignStmt{
			Lhs: []ast.Expr{m.State["lhs"].(ast.Expr)},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.Ident{Name: "append"},
					Args: []ast.Expr{
						m.State["lhs"].(ast.Expr),
						m.State["x"].(ast.Expr),
					},
					Ellipsis: 1,
				},
			},
		}

		report.NodeFG(pass, node, fmt.Sprintf("should replace loop with %s", report.Render(pass, r)),
			edit.Fix("replace loop with call to append", edit.ReplaceWithNode(pass.Fset, node, r)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.RangeStmt)(nil)}, fn)
	return nil, nil
}

var (
	checkTimeSinceQ = pattern.MustParse(`(CallExpr (SelectorExpr (CallExpr (Function "time.Now") []) (Function "(time.Time).Sub")) [arg])`)
	checkTimeSinceR = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "time") (Ident "Since")) [arg])`)
)

func CheckTimeSince(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if _, edits, ok := MatchAndEdit(pass, checkTimeSinceQ, checkTimeSinceR, node); ok {
			report.NodeFG(pass, node, "should use time.Since instead of time.Now().Sub",
				edit.Fix("replace with call to time.Since", edits...))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkTimeUntilQ = pattern.MustParse(`(CallExpr (Function "(time.Time).Sub") [(CallExpr (Function "time.Now") [])])`)
	checkTimeUntilR = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "time") (Ident "Until")) [arg])`)
)

func CheckTimeUntil(pass *analysis.Pass) (interface{}, error) {
	if !code.IsGoVersion(pass, 8) {
		return nil, nil
	}
	fn := func(node ast.Node) {
		if _, ok := Match(pass, checkTimeUntilQ, node); ok {
			if sel, ok := node.(*ast.CallExpr).Fun.(*ast.SelectorExpr); ok {
				r := pattern.NodeToAST(checkTimeUntilR.Root, map[string]interface{}{"arg": sel.X}).(ast.Node)
				report.NodeFG(pass, node, "should use time.Until instead of t.Sub(time.Now())",
					edit.Fix("replace with call to time.Until", edit.ReplaceWithNode(pass.Fset, node, r)))
			} else {
				report.NodefFG(pass, node, "should use time.Until instead of t.Sub(time.Now())")
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkUnnecessaryBlankQ1 = pattern.MustParse(`
		(AssignStmt
			[_ (Ident "_")]
			_
			(Or
				(IndexExpr _ _)
				(UnaryExpr "<-" _))) `)
	checkUnnecessaryBlankQ2 = pattern.MustParse(`
		(AssignStmt
			(Ident "_") _ recv@(UnaryExpr "<-" _))`)
)

func CheckUnnecessaryBlank(pass *analysis.Pass) (interface{}, error) {
	fn1 := func(node ast.Node) {
		if _, ok := Match(pass, checkUnnecessaryBlankQ1, node); ok {
			r := *node.(*ast.AssignStmt)
			r.Lhs = r.Lhs[0:1]
			report.NodeFG(pass, node, "unnecessary assignment to the blank identifier",
				edit.Fix("remove assignment to blank identifier", edit.ReplaceWithNode(pass.Fset, node, &r)))
		} else if m, ok := Match(pass, checkUnnecessaryBlankQ2, node); ok {
			report.NodeFG(pass, node, "unnecessary assignment to the blank identifier",
				edit.Fix("simplify channel receive operation", edit.ReplaceWithNode(pass.Fset, node, m.State["recv"].(ast.Node))))
		}
	}

	fn3 := func(node ast.Node) {
		rs := node.(*ast.RangeStmt)

		// for _
		if rs.Value == nil && code.IsBlank(rs.Key) {
			report.NodeFG(pass, rs.Key, "unnecessary assignment to the blank identifier",
				edit.Fix("remove assignment to blank identifier", edit.Delete(edit.Range{rs.Key.Pos(), rs.TokPos + 1})))
		}

		// for _, _
		if code.IsBlank(rs.Key) && code.IsBlank(rs.Value) {
			// FIXME we should mark both key and value
			report.NodeFG(pass, rs.Key, "unnecessary assignment to the blank identifier",
				edit.Fix("remove assignment to blank identifier", edit.Delete(edit.Range{rs.Key.Pos(), rs.TokPos + 1})))
		}

		// for x, _
		if !code.IsBlank(rs.Key) && code.IsBlank(rs.Value) {
			report.NodeFG(pass, rs.Value, "unnecessary assignment to the blank identifier",
				edit.Fix("remove assignment to blank identifier", edit.Delete(edit.Range{rs.Key.End(), rs.Value.End()})))
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.AssignStmt)(nil)}, fn1)
	if code.IsGoVersion(pass, 4) {
		pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.RangeStmt)(nil)}, fn3)
	}
	return nil, nil
}

func CheckSimplerStructConversion(pass *analysis.Pass) (interface{}, error) {
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
		if code.IsGoVersion(pass, 8) {
			if !types.IdenticalIgnoreTags(s1, s2) {
				return
			}
		} else {
			if !types.Identical(s1, s2) {
				return
			}
		}

		r := &ast.CallExpr{
			Fun:  lit.Type,
			Args: []ast.Expr{ident},
		}
		report.NodeFG(pass, node,
			fmt.Sprintf("should convert %s (type %s) to %s instead of using struct literal", ident.Name, typ2.Obj().Name(), typ1.Obj().Name()),
			edit.Fix("use type conversion", edit.ReplaceWithNode(pass.Fset, node, r)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.UnaryExpr)(nil), (*ast.CompositeLit)(nil)}, fn)
	return nil, nil
}

func CheckTrim(pass *analysis.Pass) (interface{}, error) {
	sameNonDynamic := func(node1, node2 ast.Node) bool {
		if reflect.TypeOf(node1) != reflect.TypeOf(node2) {
			return false
		}

		switch node1 := node1.(type) {
		case *ast.Ident:
			return node1.Obj == node2.(*ast.Ident).Obj
		case *ast.SelectorExpr:
			return report.Render(pass, node1) == report.Render(pass, node2)
		case *ast.IndexExpr:
			return report.Render(pass, node1) == report.Render(pass, node2)
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

		condCallName := code.CallNameAST(pass, condCall)
		switch condCallName {
		case "strings.HasPrefix":
			pkg = "strings"
			fun = "HasPrefix"
		case "strings.HasSuffix":
			pkg = "strings"
			fun = "HasSuffix"
		case "strings.Contains":
			pkg = "strings"
			fun = "Contains"
		case "bytes.HasPrefix":
			pkg = "bytes"
			fun = "HasPrefix"
		case "bytes.HasSuffix":
			pkg = "bytes"
			fun = "HasSuffix"
		case "bytes.Contains":
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

			rhsName := code.CallNameAST(pass, rhs)
			if condCallName == "strings.HasPrefix" && rhsName == "strings.TrimPrefix" ||
				condCallName == "strings.HasSuffix" && rhsName == "strings.TrimSuffix" ||
				condCallName == "strings.Contains" && rhsName == "strings.Replace" ||
				condCallName == "bytes.HasPrefix" && rhsName == "bytes.TrimPrefix" ||
				condCallName == "bytes.HasSuffix" && rhsName == "bytes.TrimSuffix" ||
				condCallName == "bytes.Contains" && rhsName == "bytes.Replace" {
				report.NodefFG(pass, ifstmt, "should replace this if statement with an unconditional %s", rhsName)
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
					n, ok := code.ExprToInt(pass, slice.Low)
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
					s1, ok1 := code.ExprToString(pass, lit)
					s2, ok2 := code.ExprToString(pass, condCall.Args[1])
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
				string, ok1 := code.ExprToString(pass, condCall.Args[1])
				int, ok2 := code.ExprToInt(pass, slice.Low)
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
			report.NodefFG(pass, ifstmt, "should replace this if statement with an unconditional %s.%s", pkg, replacement)
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

var (
	checkLoopSlideQ = pattern.MustParse(`
		(ForStmt
			(AssignStmt initvar@(Ident _) _ (BasicLit "INT" "0"))
			(BinaryExpr initvar "<" limit@(Ident _))
			(IncDecStmt initvar "++")
			[(AssignStmt
				(IndexExpr slice@(Ident _) initvar)
				"="
				(IndexExpr slice (BinaryExpr offset@(Ident _) "+" initvar)))])`)
	checkLoopSlideR = pattern.MustParse(`
		(CallExpr
			(Ident "copy")
			[(SliceExpr slice nil limit nil)
				(SliceExpr slice offset nil nil)])`)
)

func CheckLoopSlide(pass *analysis.Pass) (interface{}, error) {
	// TODO(dh): detect bs[i+offset] in addition to bs[offset+i]
	// TODO(dh): consider merging this function with LintLoopCopy
	// TODO(dh): detect length that is an expression, not a variable name
	// TODO(dh): support sliding to a different offset than the beginning of the slice

	fn := func(node ast.Node) {
		loop := node.(*ast.ForStmt)
		m, edits, ok := MatchAndEdit(pass, checkLoopSlideQ, checkLoopSlideR, loop)
		if !ok {
			return
		}
		if _, ok := pass.TypesInfo.TypeOf(m.State["slice"].(*ast.Ident)).Underlying().(*types.Slice); !ok {
			return
		}

		report.NodeFG(pass, loop, "should use copy() instead of loop for sliding slice elements",
			edit.Fix("use copy() instead of loop", edits...))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil)}, fn)
	return nil, nil
}

var (
	checkMakeLenCapQ1 = pattern.MustParse(`(CallExpr (Builtin "make") [typ size@(BasicLit "INT" "0")])`)
	checkMakeLenCapQ2 = pattern.MustParse(`(CallExpr (Builtin "make") [typ size size])`)
)

func CheckMakeLenCap(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkMakeLenCapQ1, node); ok {
			T := m.State["typ"].(ast.Expr)
			size := m.State["size"].(ast.Node)
			if _, ok := pass.TypesInfo.TypeOf(T).Underlying().(*types.Slice); ok {
				return
			}
			report.NodefFG(pass, size, "should use make(%s) instead", report.Render(pass, T))
		} else if m, ok := Match(pass, checkMakeLenCapQ2, node); ok {
			// TODO(dh): don't consider sizes identical if they're
			// dynamic. for example: make(T, <-ch, <-ch).
			T := m.State["typ"].(ast.Expr)
			size := m.State["size"].(ast.Node)
			report.NodefFG(pass, size,
				"should use make(%s, %s) instead", report.Render(pass, T), report.Render(pass, size))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkAssertNotNilFn1Q = pattern.MustParse(`
		(IfStmt
			(AssignStmt [(Ident "_") ok@(Object _)] _ [(TypeAssertExpr assert@(Object _) _)])
			(Or
				(BinaryExpr ok "&&" (BinaryExpr assert "!=" (Builtin "nil")))
				(BinaryExpr (BinaryExpr assert "!=" (Builtin "nil")) "&&" ok))
			_
			_)`)
	checkAssertNotNilFn2Q = pattern.MustParse(`
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
)

func CheckAssertNotNil(pass *analysis.Pass) (interface{}, error) {
	fn1 := func(node ast.Node) {
		m, ok := Match(pass, checkAssertNotNilFn1Q, node)
		if !ok {
			return
		}
		assert := m.State["assert"].(types.Object)
		assign := m.State["ok"].(types.Object)
		report.NodefFG(pass, node, "when %s is true, %s can't be nil", assign.Name(), assert.Name())
	}
	fn2 := func(node ast.Node) {
		m, ok := Match(pass, checkAssertNotNilFn2Q, node)
		if !ok {
			return
		}
		ifstmt := m.State["ifstmt"].(*ast.IfStmt)
		lhs := m.State["lhs"].(types.Object)
		assignIdent := m.State["ok"].(types.Object)
		report.NodefFG(pass, ifstmt, "when %s is true, %s can't be nil", assignIdent.Name(), lhs.Name())
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn1)
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn2)
	return nil, nil
}

func CheckDeclareAssign(pass *analysis.Pass) (interface{}, error) {
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

			r := &ast.GenDecl{
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names:  vspec.Names,
						Values: []ast.Expr{assign.Rhs[0]},
						Type:   vspec.Type,
					},
				},
				Tok: gdecl.Tok,
			}
			report.NodeFG(pass, decl, "should merge variable declaration with assignment on next line",
				edit.Fix("merge declaration with assignment", edit.ReplaceWithNode(pass.Fset, edit.Range{decl.Pos(), assign.End()}, r)))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BlockStmt)(nil)}, fn)
	return nil, nil
}

func CheckRedundantBreak(pass *analysis.Pass) (interface{}, error) {
	fn1 := func(node ast.Node) {
		clause := node.(*ast.CaseClause)
		if len(clause.Body) < 2 {
			return
		}
		branch, ok := clause.Body[len(clause.Body)-1].(*ast.BranchStmt)
		if !ok || branch.Tok != token.BREAK || branch.Label != nil {
			return
		}
		report.NodefFG(pass, branch, "redundant break statement")
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
		report.NodefFG(pass, rst, "redundant return statement")
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
	if !code.IsType(sig.Results().At(0).Type(), "string") {
		return false
	}
	return true
}

var checkRedundantSprintfQ = pattern.MustParse(`(CallExpr (Function "fmt.Sprintf") [format arg])`)

func CheckRedundantSprintf(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkRedundantSprintfQ, node)
		if !ok {
			return
		}

		format := m.State["format"].(ast.Expr)
		arg := m.State["arg"].(ast.Expr)
		if s, ok := code.ExprToString(pass, format); !ok || s != "%s" {
			return
		}
		typ := pass.TypesInfo.TypeOf(arg)

		ssapkg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).Pkg
		if types.TypeString(typ, nil) != "reflect.Value" && isStringer(typ, &ssapkg.Prog.MethodSets) {
			replacement := &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   arg,
					Sel: &ast.Ident{Name: "String"},
				},
			}
			report.Node(pass, node, "should use String() instead of fmt.Sprintf",
				edit.Fix("replace with call to String method", edit.ReplaceWithNode(pass.Fset, node, replacement)))
			return
		}

		if typ.Underlying() == types.Universe.Lookup("string").Type() {
			if typ == types.Universe.Lookup("string").Type() {
				report.NodeFG(pass, node, "the argument is already a string, there's no need to use fmt.Sprintf",
					edit.Fix("remove unnecessary call to fmt.Sprintf", edit.ReplaceWithNode(pass.Fset, node, arg)))
			} else {
				replacement := &ast.CallExpr{
					Fun:  &ast.Ident{Name: "string"},
					Args: []ast.Expr{arg},
				}
				report.NodeFG(pass, node, "the argument's underlying type is a string, should use a simple conversion instead of fmt.Sprintf",
					edit.Fix("replace with conversion to string", edit.ReplaceWithNode(pass.Fset, node, replacement)))
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkErrorsNewSprintfQ = pattern.MustParse(`(CallExpr (Function "errors.New") [(CallExpr (Function "fmt.Sprintf") args)])`)
	checkErrorsNewSprintfR = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "fmt") (Ident "Errorf")) args)`)
)

func CheckErrorsNewSprintf(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if _, edits, ok := MatchAndEdit(pass, checkErrorsNewSprintfQ, checkErrorsNewSprintfR, node); ok {
			// TODO(dh): the suggested fix may leave an unused import behind
			report.NodeFG(pass, node, "should use fmt.Errorf(...) instead of errors.New(fmt.Sprintf(...))",
				edit.Fix("use fmt.Errorf", edits...))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckRangeStringRunes(pass *analysis.Pass) (interface{}, error) {
	return sharedcheck.CheckRangeStringRunes(pass)
}

var checkNilCheckAroundRangeQ = pattern.MustParse(`
	(IfStmt
		nil
		(BinaryExpr x@(Object _) "!=" (Builtin "nil"))
		[(RangeStmt _ _ _ x _)]
		nil)`)

func CheckNilCheckAroundRange(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkNilCheckAroundRangeQ, node)
		if !ok {
			return
		}
		switch m.State["x"].(types.Object).Type().Underlying().(type) {
		case *types.Slice, *types.Map:
			report.NodeFG(pass, node, "unnecessary nil check around range")

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
	name := code.SelectorName(pass, sel)
	switch name {
	case "sort.IntSlice", "sort.Float64Slice", "sort.StringSlice":
	default:
		return true
	}

	return false
}

func CheckSortHelpers(pass *analysis.Pass) (interface{}, error) {
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
			if !code.IsCallToAST(pass, node, "sort.Sort") {
				return true
			}
			if isPermissibleSort(pass, node) {
				permissible = true
				return false
			}
			call := node.(*ast.CallExpr)
			typeconv := call.Args[Arg("sort.Sort.data")].(*ast.CallExpr)
			sel := typeconv.Fun.(*ast.SelectorExpr)
			name := code.SelectorName(pass, sel)

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
		report.NodefFG(pass, err.node, "%s", err.msg)
	}
	return nil, nil
}

var checkGuardedDeleteQ = pattern.MustParse(`
	(IfStmt
		(AssignStmt
			[(Ident "_") ok@(Ident _)]
			":="
			(IndexExpr m key))
		ok
		[call@(CallExpr (Builtin "delete") [m key])]
		nil)`)

func CheckGuardedDelete(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkGuardedDeleteQ, node); ok {
			report.Report(pass, report.Options{
				Node:            node,
				ShortRange:      true,
				FilterGenerated: true,
				Message:         "unnecessary guard around call to delete",
				Fixes: []analysis.SuggestedFix{
					edit.Fix("remove guard", edit.ReplaceWithNode(pass.Fset, node, m.State["call"].(ast.Node))),
				},
			})
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

var (
	checkSimplifyTypeSwitchQ = pattern.MustParse(`
		(TypeSwitchStmt
			nil
			expr@(TypeAssertExpr ident@(Ident _) _)
			body)`)
	checkSimplifyTypeSwitchR = pattern.MustParse(`(AssignStmt ident ":=" expr)`)
)

func CheckSimplifyTypeSwitch(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkSimplifyTypeSwitchQ, node)
		if !ok {
			return
		}
		stmt := node.(*ast.TypeSwitchStmt)
		expr := m.State["expr"].(ast.Node)
		ident := m.State["ident"].(*ast.Ident)

		x := pass.TypesInfo.ObjectOf(ident)
		var allOffenders []*ast.TypeAssertExpr
		canSuggestFix := true
		for _, clause := range stmt.Body.List {
			clause := clause.(*ast.CaseClause)
			if len(clause.List) != 1 {
				continue
			}
			hasUnrelatedAssertion := false
			var offenders []*ast.TypeAssertExpr
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
			canSuggestFix = canSuggestFix && !hasUnrelatedAssertion
		}
		if len(allOffenders) != 0 {
			at := ""
			for _, offender := range allOffenders {
				pos := lint.DisplayPosition(pass.Fset, offender.Pos())
				at += "\n\t" + pos.String()
			}

			msg := fmt.Sprintf("assigning the result of this type assertion to a variable (switch %s := %s.(type)) could eliminate the following type assertions:%s",
				report.Render(pass, ident), report.Render(pass, ident), at)
			if canSuggestFix {
				var edits []analysis.TextEdit
				edits = append(edits, edit.ReplaceWithPattern(pass, checkSimplifyTypeSwitchR, m.State, expr))
				for _, offender := range allOffenders {
					edits = append(edits, edit.ReplaceWithNode(pass.Fset, offender, offender.X))
				}
				report.NodeFG(pass, expr, msg, edit.Fix("simplify type switch", edits...))
			} else {
				report.NodeFG(pass, expr, msg)
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.TypeSwitchStmt)(nil)}, fn)
	return nil, nil
}

func CheckRedundantCanonicalHeaderKey(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		callName := code.CallNameAST(pass, call)
		switch callName {
		case "(net/http.Header).Add", "(net/http.Header).Del", "(net/http.Header).Get", "(net/http.Header).Set":
		default:
			return
		}

		if !code.IsCallToAST(pass, call.Args[0], "net/http.CanonicalHeaderKey") {
			return
		}

		report.NodeFG(pass, call,
			fmt.Sprintf("calling net/http.CanonicalHeaderKey on the 'key' argument of %s is redundant", callName),
			edit.Fix("remove call to CanonicalHeaderKey", edit.ReplaceWithNode(pass.Fset, call.Args[0], call.Args[0].(*ast.CallExpr).Args[0])))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var checkUnnecessaryGuardQ = pattern.MustParse(`
	(Or
		(IfStmt
			(AssignStmt [(Ident "_") ok@(Ident _)] ":=" indexexpr@(IndexExpr _ _))
			ok
			set@(AssignStmt indexexpr "=" (CallExpr (Builtin "append") indexexpr:values))
			(AssignStmt indexexpr "=" (CompositeLit _ values)))
		(IfStmt
			(AssignStmt [(Ident "_") ok] ":=" indexexpr@(IndexExpr _ _))
			ok
			set@(AssignStmt indexexpr "+=" value)
			(AssignStmt indexexpr "=" value))
		(IfStmt
			(AssignStmt [(Ident "_") ok] ":=" indexexpr@(IndexExpr _ _))
			ok
			set@(IncDecStmt indexexpr "++")
			(AssignStmt indexexpr "=" (BasicLit "INT" "1"))))`)

func CheckUnnecessaryGuard(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkUnnecessaryGuardQ, node); ok {
			if code.MayHaveSideEffects(m.State["indexexpr"].(ast.Expr)) {
				return
			}
			report.Node(pass, node, "unnecessary guard around map access",
				edit.Fix("simplify map access", edit.ReplaceWithNode(pass.Fset, node, m.State["set"].(ast.Node))))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

var (
	checkElaborateSleepQ = pattern.MustParse(`(SelectStmt (CommClause (UnaryExpr "<-" (CallExpr (Function "time.After") [arg])) body))`)
	checkElaborateSleepR = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "time") (Ident "Sleep")) [arg])`)
)

func CheckElaborateSleep(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkElaborateSleepQ, node); ok {
			if body, ok := m.State["body"].([]ast.Stmt); ok && len(body) == 0 {
				report.NodeFG(pass, node, "should use time.Sleep instead of elaborate way of sleeping",
					edit.Fix("Use time.Sleep", edit.ReplaceWithPattern(pass, checkElaborateSleepR, m.State, node)))
			} else {
				// TODO(dh): we could make a suggested fix if the body
				// doesn't declare or shadow any identifiers
				report.NodefFG(pass, node, "should use time.Sleep instead of elaborate way of sleeping")
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.SelectStmt)(nil)}, fn)
	return nil, nil
}
