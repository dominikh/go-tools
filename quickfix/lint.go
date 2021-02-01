package quickfix

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"honnef.co/go/tools/analysis/code"
	"honnef.co/go/tools/analysis/edit"
	"honnef.co/go/tools/analysis/report"
	"honnef.co/go/tools/go/ast/astutil"
	"honnef.co/go/tools/go/types/typeutil"
	"honnef.co/go/tools/pattern"

	"golang.org/x/tools/go/analysis"
)

var (
	stringsIndexQ = pattern.MustParse(`
		(CallExpr
			fn@(Or
				(Function "strings.Index")
				(Function "strings.LastIndex")
				(Function "strings.IndexByte")
				(Function "strings.LastIndexByte"))
			[arg1 lit@(BasicLit (Or "STRING" "CHAR") _)])`)
	bytesIndexQ = pattern.MustParse(`
		(CallExpr
			fn@(Or
				(Function "bytes.Index")
				(Function "bytes.LastIndex"))
			[
				arg1
				(Or
					(CompositeLit _ [b])
					(CallExpr (ArrayType nil _) lit@(BasicLit "STRING" _)))])`)

	stringsIndexR     = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "strings") (Ident replacement)) [arg1 (BasicLit "CHAR" lit)])`)
	stringsIndexByteR = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "strings") (Ident replacement)) [arg1 (BasicLit "STRING" lit)])`)
	bytesIndexR       = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "bytes") (Ident replacement)) [arg1 b])`)
)

func CheckStringsIndexByte(pass *analysis.Pass) (interface{}, error) {
	// FIXME(dh): create proper suggested fix for renamed import

	fn := func(node ast.Node) {
		if matcher, ok := code.Match(pass, stringsIndexQ, node); ok {
			var replacement string
			isString := false
			fn := typeutil.FuncName(matcher.State["fn"].(*types.Func))
			switch fn {
			case "strings.Index":
				replacement = "IndexByte"
				isString = true
			case "strings.LastIndex":
				replacement = "LastIndexByte"
				isString = true
			case "strings.IndexByte":
				replacement = "Index"
			case "strings.LastIndexByte":
				replacement = "LastIndex"
			}

			var rPattern pattern.Pattern
			lit := matcher.State["lit"].(*ast.BasicLit).Value
			var newLit string
			if isString {
				if q, err := strconv.Unquote(lit); err != nil || len(q) != 1 {
					return
				}
				newLit = "'" + lit[1:len(lit)-1] + "'"
				rPattern = stringsIndexR
			} else {
				newLit = `"` + lit[1:len(lit)-1] + `"`
				rPattern = stringsIndexByteR
			}

			state := pattern.State{
				"arg1":        matcher.State["arg1"],
				"lit":         newLit,
				"replacement": replacement,
			}
			report.Report(pass, node, fmt.Sprintf("could use strings.%s instead of %s", replacement, fn),
				report.Fixes(edit.Fix(fmt.Sprintf("Use strings.%s instead of %s", replacement, fn), edit.ReplaceWithPattern(pass, rPattern, state, node))))
		} else if matcher, ok := code.Match(pass, bytesIndexQ, node); ok {
			var replacement string
			fn := typeutil.FuncName(matcher.State["fn"].(*types.Func))
			switch fn {
			case "bytes.Index":
				replacement = "IndexByte"
			case "bytes.LastIndex":
				replacement = "LastIndexByte"
			}

			if _, ok := matcher.State["b"]; ok {
				state := pattern.State{
					"arg1":        matcher.State["arg1"],
					"b":           matcher.State["b"],
					"replacement": replacement,
				}
				report.Report(pass, node, fmt.Sprintf("could use bytes.%s instead of %s", replacement, fn),
					report.Fixes(edit.Fix(fmt.Sprintf("Use bytes.%s instead of %s", replacement, fn), edit.ReplaceWithPattern(pass, bytesIndexR, state, node))))
			} else {
				lit := matcher.State["lit"].(*ast.BasicLit).Value
				if q, err := strconv.Unquote(lit); err != nil || len(q) != 1 {
					return
				}
				state := pattern.State{
					"arg1": matcher.State["arg1"],
					"b": &ast.BasicLit{
						Kind:  token.CHAR,
						Value: "'" + lit[1:len(lit)-1] + "'",
					},
					"replacement": replacement,
				}
				report.Report(pass, node, fmt.Sprintf("could use bytes.%s instead of %s", replacement, fn),
					report.Fixes(edit.Fix(fmt.Sprintf("Use bytes.%s instead of %s", replacement, fn), edit.ReplaceWithPattern(pass, bytesIndexR, state, node))))
			}
		}
	}
	code.Preorder(pass, fn, (*ast.CallExpr)(nil))
	return nil, nil
}

func negateDeMorgan(expr ast.Expr, recursive bool) ast.Expr {
	switch expr := expr.(type) {
	case *ast.BinaryExpr:
		var out ast.BinaryExpr
		switch expr.Op {
		case token.EQL:
			out.X = expr.X
			out.Op = token.NEQ
			out.Y = expr.Y
		case token.LSS:
			out.X = expr.X
			out.Op = token.GEQ
			out.Y = expr.Y
		case token.GTR:
			out.X = expr.X
			out.Op = token.LEQ
			out.Y = expr.Y
		case token.NEQ:
			out.X = expr.X
			out.Op = token.EQL
			out.Y = expr.Y
		case token.LEQ:
			out.X = expr.X
			out.Op = token.GTR
			out.Y = expr.Y
		case token.GEQ:
			out.X = expr.X
			out.Op = token.LSS
			out.Y = expr.Y

		case token.LAND:
			out.X = negateDeMorgan(expr.X, recursive)
			out.Op = token.LOR
			out.Y = negateDeMorgan(expr.Y, recursive)
		case token.LOR:
			out.X = negateDeMorgan(expr.X, recursive)
			out.Op = token.LAND
			out.Y = negateDeMorgan(expr.Y, recursive)
		}
		return &out

	case *ast.ParenExpr:
		if recursive {
			return &ast.ParenExpr{
				X: negateDeMorgan(expr.X, recursive),
			}
		} else {
			return &ast.UnaryExpr{
				Op: token.NOT,
				X:  expr,
			}
		}

	case *ast.UnaryExpr:
		if expr.Op == token.NOT {
			return expr.X
		} else {
			return &ast.UnaryExpr{
				Op: token.NOT,
				X:  expr,
			}
		}

	default:
		return &ast.UnaryExpr{
			Op: token.NOT,
			X:  expr,
		}
	}
}

func simplifyParentheses(node ast.Expr) ast.Expr {
	var changed bool
	// XXX accept list of ops to operate on
	// XXX copy AST node, don't modify in place
	post := func(c *astutil.Cursor) bool {
		out := c.Node()
		if paren, ok := c.Node().(*ast.ParenExpr); ok {
			out = paren.X
		}

		if binop, ok := out.(*ast.BinaryExpr); ok {
			if right, ok := binop.Y.(*ast.BinaryExpr); ok && binop.Op == right.Op {
				// XXX also check that Op is associative

				root := binop
				pivot := root.Y.(*ast.BinaryExpr)
				root.Y = pivot.X
				pivot.X = root
				root = pivot
				out = root
			}
		}

		if out != c.Node() {
			changed = true
			c.Replace(out)
		}
		return true
	}

	for changed = true; changed; {
		changed = false
		node = astutil.Apply(node, nil, post).(ast.Expr)
	}

	return node
}

var demorganQ = pattern.MustParse(`(UnaryExpr "!" expr@(BinaryExpr _ _ _))`)

func CheckDeMorgan(pass *analysis.Pass) (interface{}, error) {
	// TODO(dh): support going in the other direction, e.g. turning `!a && !b && !c` into `!(a || b || c)`

	// hasFloats reports whether any subexpression is of type float.
	hasFloats := func(expr ast.Expr) bool {
		found := false
		ast.Inspect(expr, func(node ast.Node) bool {
			if expr, ok := node.(ast.Expr); ok {
				if basic, ok := pass.TypesInfo.TypeOf(expr).Underlying().(*types.Basic); ok {
					if (basic.Info() & types.IsFloat) != 0 {
						found = true
						return false
					}
				}
			}
			return true
		})
		return found
	}

	fn := func(node ast.Node, stack []ast.Node) {
		matcher, ok := code.Match(pass, demorganQ, node)
		if !ok {
			return
		}

		expr := matcher.State["expr"].(ast.Expr)

		// be extremely conservative when it comes to floats
		if hasFloats(expr) {
			return
		}

		n := negateDeMorgan(expr, false)
		nr := negateDeMorgan(expr, true)
		ns := simplifyParentheses(astutil.CopyExpr(n))
		nrs := simplifyParentheses(astutil.CopyExpr(nr))

		var bn, bnr, bns, bnrs string
		switch parent := stack[len(stack)-2]; parent.(type) {
		case *ast.BinaryExpr, *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt:
			// Always add parentheses for if, for and switch. If
			// they're unnecessary, go/printer will strip them when
			// the whole file gets formatted.

			bn = report.Render(pass, &ast.ParenExpr{X: n})
			bnr = report.Render(pass, &ast.ParenExpr{X: nr})
			bns = report.Render(pass, &ast.ParenExpr{X: ns})
			bnrs = report.Render(pass, &ast.ParenExpr{X: nrs})

		default:
			// TODO are there other types where we don't want to strip parentheses?
			bn = report.Render(pass, n)
			bnr = report.Render(pass, nr)
			bns = report.Render(pass, ns)
			bnrs = report.Render(pass, nrs)
		}

		// Note: we cannot compare the ASTs directly, because
		// simplifyParentheses might have rebalanced trees without
		// affecting the rendered form.
		var fixes []analysis.SuggestedFix
		fixes = append(fixes, edit.Fix("Apply De Morgan's law", edit.ReplaceWithString(pass.Fset, node, bn)))
		if bn != bns {
			fixes = append(fixes, edit.Fix("Apply De Morgan's law & simplify", edit.ReplaceWithString(pass.Fset, node, bns)))
		}
		if bn != bnr {
			fixes = append(fixes, edit.Fix("Apply De Morgan's law recursively", edit.ReplaceWithString(pass.Fset, node, bnr)))
			if bnr != bnrs {
				fixes = append(fixes, edit.Fix("Apply De Morgan's law recursively & simplify", edit.ReplaceWithString(pass.Fset, node, bnrs)))
			}
		}

		report.Report(pass, node, "could apply De Morgan's law", report.Fixes(fixes...))
	}

	code.PreorderStack(pass, fn, (*ast.UnaryExpr)(nil))

	return nil, nil
}

func findSwitchPairs(pass *analysis.Pass, expr ast.Expr, pairs *[]*ast.BinaryExpr) (OUT bool) {
	binexpr, ok := astutil.Unparen(expr).(*ast.BinaryExpr)
	if !ok {
		return false
	}
	switch binexpr.Op {
	case token.EQL:
		if code.MayHaveSideEffects(pass, binexpr.X, nil) || code.MayHaveSideEffects(pass, binexpr.Y, nil) {
			return false
		}
		// syntactic identity should suffice. we do not allow side
		// effects in the case clauses, so there should be no way for
		// values to change.
		if len(*pairs) > 0 && !astutil.Equal(binexpr.X, (*pairs)[0].X) {
			return false
		}
		*pairs = append(*pairs, binexpr)
		return true
	case token.LOR:
		return findSwitchPairs(pass, binexpr.X, pairs) && findSwitchPairs(pass, binexpr.Y, pairs)
	default:
		return false
	}
}

func CheckTaglessSwitch(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		swtch := node.(*ast.SwitchStmt)
		if swtch.Tag != nil || len(swtch.Body.List) == 0 {
			return
		}

		pairs := make([][]*ast.BinaryExpr, len(swtch.Body.List))
		for i, stmt := range swtch.Body.List {
			stmt := stmt.(*ast.CaseClause)
			for _, cond := range stmt.List {
				if !findSwitchPairs(pass, cond, &pairs[i]) {
					return
				}
			}
		}

		var x ast.Expr
		for _, pair := range pairs {
			if len(pair) == 0 {
				continue
			}
			if x == nil {
				x = pair[0].X
			} else {
				if !astutil.Equal(x, pair[0].X) {
					return
				}
			}
		}
		if x == nil {
			// the switch only has a default case
			if len(pairs) > 1 {
				panic("found more than one case clause with no pairs")
			}
			return
		}

		edits := make([]analysis.TextEdit, 0, len(swtch.Body.List)+1)
		for i, stmt := range swtch.Body.List {
			stmt := stmt.(*ast.CaseClause)
			if stmt.List == nil {
				continue
			}

			var values []string
			for _, binexpr := range pairs[i] {
				y := binexpr.Y
				if p, ok := y.(*ast.ParenExpr); ok {
					y = p.X
				}
				values = append(values, report.Render(pass, y))
			}

			edits = append(edits, edit.ReplaceWithString(pass.Fset, edit.Range{stmt.List[0].Pos(), stmt.Colon}, strings.Join(values, ", ")))
		}
		pos := swtch.Switch + token.Pos(len("switch"))
		edits = append(edits, edit.ReplaceWithString(pass.Fset, edit.Range{pos, pos}, " "+report.Render(pass, x)))
		report.Report(pass, swtch, fmt.Sprintf("could use tagged switch on %s", report.Render(pass, x)),
			report.Fixes(edit.Fix("Replace with tagged switch", edits...)))
	}

	code.Preorder(pass, fn, (*ast.SwitchStmt)(nil))
	return nil, nil
}

func CheckIfElseToSwitch(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node, stack []ast.Node) {
		if _, ok := stack[len(stack)-2].(*ast.IfStmt); ok {
			// this if statement is part of an if-else chain
			return
		}
		ifstmt := node.(*ast.IfStmt)

		m := map[ast.Expr][]*ast.BinaryExpr{}
		for item := ifstmt; item != nil; {
			if item.Init != nil {
				return
			}
			if item.Body == nil {
				return
			}

			skip := false
			ast.Inspect(item.Body, func(node ast.Node) bool {
				if branch, ok := node.(*ast.BranchStmt); ok && branch.Tok != token.GOTO {
					skip = true
					return false
				}
				return true
			})
			if skip {
				return
			}

			var pairs []*ast.BinaryExpr
			if !findSwitchPairs(pass, item.Cond, &pairs) {
				return
			}
			m[item.Cond] = pairs
			switch els := item.Else.(type) {
			case *ast.IfStmt:
				item = els
			case *ast.BlockStmt, nil:
				item = nil
			default:
				panic(fmt.Sprintf("unreachable: %T", els))
			}
		}

		var x ast.Expr
		for _, pair := range m {
			if len(pair) == 0 {
				continue
			}
			if x == nil {
				x = pair[0].X
			} else {
				if !astutil.Equal(x, pair[0].X) {
					return
				}
			}
		}
		if x == nil {
			// shouldn't happen
			return
		}

		// We require at least two 'if' to make this suggestion, to
		// avoid clutter in the editor.
		if len(m) < 2 {
			return
		}

		var edits []analysis.TextEdit
		for item := ifstmt; item != nil; {
			var end token.Pos
			if item.Else != nil {
				end = item.Else.Pos()
			} else {
				// delete up to but not including the closing brace.
				end = item.Body.Rbrace
			}

			var conds []string
			for _, cond := range m[item.Cond] {
				y := cond.Y
				if p, ok := y.(*ast.ParenExpr); ok {
					y = p.X
				}
				conds = append(conds, report.Render(pass, y))
			}
			sconds := strings.Join(conds, ", ")
			edits = append(edits,
				edit.ReplaceWithString(pass.Fset, edit.Range{item.If, item.Body.Lbrace + 1}, "case "+sconds+":"),
				edit.Delete(edit.Range{item.Body.Rbrace, end}))

			switch els := item.Else.(type) {
			case *ast.IfStmt:
				item = els
			case *ast.BlockStmt:
				edits = append(edits, edit.ReplaceWithString(pass.Fset, edit.Range{els.Lbrace, els.Lbrace + 1}, "default:"))
				item = nil
			case nil:
				item = nil
			default:
				panic(fmt.Sprintf("unreachable: %T", els))
			}
		}
		// FIXME this forces the first case to begin in column 0. try to fix the indentation
		edits = append(edits, edit.ReplaceWithString(pass.Fset, edit.Range{ifstmt.If, ifstmt.If}, fmt.Sprintf("switch %s {\n", report.Render(pass, x))))
		report.Report(pass, ifstmt, fmt.Sprintf("could use tagged switch on %s", report.Render(pass, x)),
			report.Fixes(edit.Fix("Replace with tagged switch", edits...)))
	}
	code.PreorderStack(pass, fn, (*ast.IfStmt)(nil))
	return nil, nil
}

var stringsReplaceAllQ = pattern.MustParse(`(Or
	(CallExpr fn@(Function "strings.Replace") [_ _ _ lit@(UnaryExpr "-" (BasicLit "INT" "1"))])
	(CallExpr fn@(Function "strings.SplitN") [_ _ lit@(UnaryExpr "-" (BasicLit "INT" "1"))])
	(CallExpr fn@(Function "strings.SplitAfterN") [_ _ lit@(UnaryExpr "-" (BasicLit "INT" "1"))])
	(CallExpr fn@(Function "bytes.Replace") [_ _ _ lit@(UnaryExpr "-" (BasicLit "INT" "1"))])
	(CallExpr fn@(Function "bytes.SplitN") [_ _ lit@(UnaryExpr "-" (BasicLit "INT" "1"))])
	(CallExpr fn@(Function "bytes.SplitAfterN") [_ _ lit@(UnaryExpr "-" (BasicLit "INT" "1"))]))`)

func CheckStringsReplaceAll(pass *analysis.Pass) (interface{}, error) {
	// XXX respect minimum Go version

	// FIXME(dh): create proper suggested fix for renamed import

	fn := func(node ast.Node) {
		matcher, ok := code.Match(pass, stringsReplaceAllQ, node)
		if !ok {
			return
		}

		var replacement string
		switch typeutil.FuncName(matcher.State["fn"].(*types.Func)) {
		case "strings.Replace":
			replacement = "strings.ReplaceAll"
		case "strings.SplitN":
			replacement = "strings.Split"
		case "strings.SplitAfterN":
			replacement = "strings.SplitAfter"
		case "bytes.Replace":
			replacement = "bytes.ReplaceAll"
		case "bytes.SplitN":
			replacement = "bytes.Split"
		case "bytes.SplitAfterN":
			replacement = "bytes.SplitAfter"
		default:
			panic("unreachable")
		}

		call := node.(*ast.CallExpr)
		report.Report(pass, call.Fun, fmt.Sprintf("could use %s instead", replacement),
			report.Fixes(edit.Fix(fmt.Sprintf("Use %s instead", replacement),
				edit.ReplaceWithString(pass.Fset, call.Fun, replacement),
				edit.Delete(matcher.State["lit"].(ast.Node)))))
	}
	code.Preorder(pass, fn, (*ast.CallExpr)(nil))
	return nil, nil
}

var mathPowQ = pattern.MustParse(`(CallExpr (Function "math.Pow") [x n@(BasicLit "INT" _)])`)

func CheckMathPow(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		matcher, ok := code.Match(pass, mathPowQ, node)
		if !ok {
			return
		}

		x := matcher.State["x"].(ast.Expr)
		if code.MayHaveSideEffects(pass, x, nil) {
			return
		}
		n, _ := strconv.ParseInt(matcher.State["n"].(*ast.BasicLit).Value, 10, 64)

		needConversion := false
		if T, ok := pass.TypesInfo.Types[x]; ok && T.Value != nil {
			info := types.Info{
				Types: map[ast.Expr]types.TypeAndValue{},
			}

			// determine if the constant expression would have type float64 if used on its own
			if err := types.CheckExpr(pass.Fset, pass.Pkg, x.Pos(), x, &info); err != nil {
				// This should not happen
				return
			}
			if T, ok := info.Types[x].Type.(*types.Basic); ok {
				if T.Kind() != types.UntypedFloat && T.Kind() != types.Float64 {
					needConversion = true
				}
			} else {
				needConversion = true
			}
		}

		var replacement ast.Expr
		switch n {
		case 0:
			replacement = &ast.BasicLit{
				Kind:  token.FLOAT,
				Value: "1.0",
			}
		case 1:
			replacement = x
		case 2, 3:
			r := &ast.BinaryExpr{
				X:  x,
				Op: token.MUL,
				Y:  x,
			}
			for i := 3; i <= int(n); i++ {
				r = &ast.BinaryExpr{
					X:  r,
					Op: token.MUL,
					Y:  x,
				}
			}

			replacement = simplifyParentheses(astutil.CopyExpr(r))
		default:
			return
		}
		if needConversion && n != 0 {
			replacement = &ast.CallExpr{
				Fun:  &ast.Ident{Name: "float64"},
				Args: []ast.Expr{replacement},
			}
		}
		report.Report(pass, node, "could expand call to math.Pow",
			report.Fixes(edit.Fix("Expand call to math.Pow", edit.ReplaceWithNode(pass.Fset, node, replacement))))
	}
	code.Preorder(pass, fn, (*ast.CallExpr)(nil))
	return nil, nil
}

var checkForLoopIfBreak = pattern.MustParse(`(ForStmt nil nil nil if@(IfStmt nil cond (BranchStmt "BREAK" nil) nil):_)`)

func CheckForLoopIfBreak(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := code.Match(pass, checkForLoopIfBreak, node)
		if !ok {
			return
		}

		pos := node.Pos() + token.Pos(len("for"))
		r := negateDeMorgan(m.State["cond"].(ast.Expr), false)

		// FIXME(dh): we're leaving behind an empty line when we
		// delete the old if statement. However, we can't just delete
		// an additional character, in case there closing curly brace
		// is followed by a comment, or Windows newlines.
		report.Report(pass, m.State["if"].(ast.Node), "could lift into loop condition",
			report.Fixes(edit.Fix("Lift into loop condition",
				edit.ReplaceWithString(pass.Fset, edit.Range{pos, pos}, " "+report.Render(pass, r)),
				edit.Delete(m.State["if"].(ast.Node)))))
	}
	code.Preorder(pass, fn, (*ast.ForStmt)(nil))
	return nil, nil
}

var checkConditionalAssignmentQ = pattern.MustParse(`(AssignStmt x@(Object _) ":=" assign@(Builtin b@(Or "true" "false")))`)
var checkConditionalAssignmentIfQ = pattern.MustParse(`(IfStmt nil cond [(AssignStmt x@(Object _) "=" (Builtin b@(Or "true" "false")))] nil)`)

func CheckConditionalAssignment(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		var body *ast.BlockStmt
		switch node := node.(type) {
		case *ast.FuncDecl:
			body = node.Body
		case *ast.FuncLit:
			body = node.Body
		default:
			panic("unreachable")
		}
		if body == nil {
			return
		}

		stmts := body.List
		if len(stmts) < 2 {
			return
		}
		for i, first := range stmts[:len(stmts)-1] {
			second := stmts[i+1]
			m1, ok := code.Match(pass, checkConditionalAssignmentQ, first)
			if !ok {
				continue
			}
			m2, ok := code.Match(pass, checkConditionalAssignmentIfQ, second)
			if !ok {
				continue
			}
			if m1.State["x"] != m2.State["x"] {
				continue
			}
			if m1.State["b"] == m2.State["b"] {
				continue
			}

			v := m2.State["cond"].(ast.Expr)
			if m1.State["b"] == "true" {
				v = &ast.UnaryExpr{
					Op: token.NOT,
					X:  v,
				}
			}
			report.Report(pass, first, "could merge conditional assignment into variable declaration",
				report.Fixes(edit.Fix("Merge conditional assignment into variable declaration",
					edit.ReplaceWithNode(pass.Fset, m1.State["assign"].(ast.Node), v),
					edit.Delete(second))))
		}
	}
	code.Preorder(pass, fn, (*ast.FuncDecl)(nil), (*ast.FuncLit)(nil))
	return nil, nil
}
