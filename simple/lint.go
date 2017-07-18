// Package simple contains a linter for Go source code.
package simple // import "honnef.co/go/tools/simple"

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"reflect"
	"strconv"
	"strings"

	"honnef.co/go/tools/internal/sharedcheck"
	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/ssa"

	"golang.org/x/tools/go/types/typeutil"
)

type Checker struct {
	CheckGenerated bool
	MS             *typeutil.MethodSetCache

	nodeFns map[ast.Node]*ssa.Function
}

func NewChecker() *Checker {
	return &Checker{
		MS: &typeutil.MethodSetCache{},
	}
}

func (c *Checker) Init(prog *lint.Program) {
	c.nodeFns = lint.NodeFns(prog.Packages)
}

func (c *Checker) Funcs() map[string]lint.Func {
	return map[string]lint.Func{
		"S1000": c.LintSingleCaseSelect,
		"S1001": c.LintLoopCopy,
		"S1002": c.LintIfBoolCmp,
		"S1003": c.LintStringsContains,
		"S1004": c.LintBytesCompare,
		"S1005": c.LintRanges,
		"S1006": c.LintForTrue,
		"S1007": c.LintRegexpRaw,
		"S1008": c.LintIfReturn,
		"S1009": c.LintRedundantNilCheckWithLen,
		"S1010": c.LintSlicing,
		"S1011": c.LintLoopAppend,
		"S1012": c.LintTimeSince,
		"S1013": c.LintSimplerReturn,
		"S1014": c.LintReceiveIntoBlank,
		"S1015": c.LintFormatInt,
		"S1016": c.LintSimplerStructConversion,
		"S1017": c.LintTrim,
		"S1018": c.LintLoopSlide,
		"S1019": c.LintMakeLenCap,
		"S1020": c.LintAssertNotNil,
		"S1021": c.LintDeclareAssign,
		"S1022": c.LintBlankOK,
		"S1023": c.LintRedundantBreak,
		"S1024": c.LintTimeUntil,
		"S1025": c.LintRedundantSprintf,
		"S1026": c.LintStringCopy,
		"S1027": c.LintRedundantReturn,
		"S1028": c.LintErrorsNewSprintf,
		"S1029": c.LintRangeStringRunes,
		"S1030": c.LintBytesConversion,
	}
}

func (c *Checker) filterGenerated(files []*ast.File) []*ast.File {
	if c.CheckGenerated {
		return files
	}
	var out []*ast.File
	for _, f := range files {
		if !lint.IsGenerated(f) {
			out = append(out, f)
		}
	}
	return out
}

func (c *Checker) LintSingleCaseSelect(j *lint.Job) {
	isSingleSelect := func(node ast.Node) bool {
		v, ok := node.(*ast.SelectStmt)
		if !ok {
			return false
		}
		return len(v.Body.List) == 1
	}

	seen := map[ast.Node]struct{}{}
	fn := func(node ast.Node) bool {
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
			j.Errorf(node, "should use for range instead of for { select {} }")
		case *ast.SelectStmt:
			if _, ok := seen[v]; ok {
				return true
			}
			if !isSingleSelect(v) {
				return true
			}
			j.Errorf(node, "should use a simple channel send/receive instead of select with a single case")
			return true
		}
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintLoopCopy(j *lint.Job) {
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
		if _, ok := j.Program.Info.TypeOf(lhs.X).(*types.Slice); !ok {
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
		if j.Program.Info.TypeOf(lhs) == nil || j.Program.Info.TypeOf(stmt.Rhs[0]) == nil {
			return true
		}
		if j.Program.Info.ObjectOf(lidx) != j.Program.Info.ObjectOf(key) {
			return true
		}
		if !types.Identical(j.Program.Info.TypeOf(lhs), j.Program.Info.TypeOf(stmt.Rhs[0])) {
			return true
		}
		if _, ok := j.Program.Info.TypeOf(loop.X).(*types.Slice); !ok {
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
			if j.Program.Info.ObjectOf(ridx) != j.Program.Info.ObjectOf(key) {
				return true
			}
		} else if rhs, ok := stmt.Rhs[0].(*ast.Ident); ok {
			value, ok := loop.Value.(*ast.Ident)
			if !ok {
				return true
			}
			if j.Program.Info.ObjectOf(rhs) != j.Program.Info.ObjectOf(value) {
				return true
			}
		} else {
			return true
		}
		j.Errorf(loop, "should use copy() instead of a loop")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintIfBoolCmp(j *lint.Job) {
	fn := func(node ast.Node) bool {
		expr, ok := node.(*ast.BinaryExpr)
		if !ok || (expr.Op != token.EQL && expr.Op != token.NEQ) {
			return true
		}
		x := j.IsBoolConst(expr.X)
		y := j.IsBoolConst(expr.Y)
		if !x && !y {
			return true
		}
		var other ast.Expr
		var val bool
		if x {
			val = j.BoolConst(expr.X)
			other = expr.Y
		} else {
			val = j.BoolConst(expr.Y)
			other = expr.X
		}
		basic, ok := j.Program.Info.TypeOf(other).Underlying().(*types.Basic)
		if !ok || basic.Kind() != types.Bool {
			return true
		}
		op := ""
		if (expr.Op == token.EQL && !val) || (expr.Op == token.NEQ && val) {
			op = "!"
		}
		r := op + j.Render(other)
		l1 := len(r)
		r = strings.TrimLeft(r, "!")
		if (l1-len(r))%2 == 1 {
			r = "!" + r
		}
		j.Errorf(expr, "should omit comparison to bool constant, can be simplified to %s", r)
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintBytesConversion(j *lint.Job) {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}

		var castName string
		switch typ := call.Fun.(type) {
		case *ast.Ident:
			if j.Program.Info.ObjectOf(typ).Type() != types.Universe.Lookup("string").Type() {
				return true
			}
			castName = "string"
		case *ast.ArrayType:
			arrTyp, ok := typ.Elt.(*ast.Ident)
			if !ok {
				return true
			}
			if j.Program.Info.ObjectOf(arrTyp).Type() != types.Universe.Lookup("byte").Type() {
				return true
			}
			castName = "[]byte"
		default:
			return true
		}

		argCall, ok := call.Args[0].(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := argCall.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		switch {
		case castName == "string" && j.IsCallToAST(call.Args[0], "(*bytes.Buffer).Bytes"):
			j.Errorf(call, "should use %v.String() instead of %v", j.Render(sel.X), j.Render(call))
		case castName == "[]byte" && j.IsCallToAST(call.Args[0], "(*bytes.Buffer).String"):
			j.Errorf(call, "should use %v.Bytes() instead of %v", j.Render(sel.X), j.Render(call))
		}
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintStringsContains(j *lint.Job) {
	// map of value to token to bool value
	allowed := map[int64]map[token.Token]bool{
		-1: {token.GTR: true, token.NEQ: true, token.EQL: false},
		0:  {token.GEQ: true, token.LSS: false},
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

		value, ok := j.ExprToInt(expr.Y)
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
		j.Errorf(node, "should use %s%s.%s(%s) instead", prefix, pkgIdent.Name, newFunc, j.RenderArgs(call.Args))

		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintBytesCompare(j *lint.Job) {
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
		if !j.IsCallToAST(call, "bytes.Compare") {
			return true
		}
		value, ok := j.ExprToInt(expr.Y)
		if !ok || value != 0 {
			return true
		}
		args := j.RenderArgs(call.Args)
		prefix := ""
		if expr.Op == token.NEQ {
			prefix = "!"
		}
		j.Errorf(node, "should use %sbytes.Equal(%s) instead", prefix, args)
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintRanges(j *lint.Job) {
	fn := func(node ast.Node) bool {
		rs, ok := node.(*ast.RangeStmt)
		if !ok {
			return true
		}
		if lint.IsBlank(rs.Key) && (rs.Value == nil || lint.IsBlank(rs.Value)) {
			j.Errorf(rs.Key, "should omit values from range; this loop is equivalent to `for range ...`")
		}

		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintForTrue(j *lint.Job) {
	fn := func(node ast.Node) bool {
		loop, ok := node.(*ast.ForStmt)
		if !ok {
			return true
		}
		if loop.Init != nil || loop.Post != nil {
			return true
		}
		if !j.IsBoolConst(loop.Cond) || !j.BoolConst(loop.Cond) {
			return true
		}
		j.Errorf(loop, "should use for {} instead of for true {}")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintRegexpRaw(j *lint.Job) {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !j.IsCallToAST(call, "regexp.MustCompile") &&
			!j.IsCallToAST(call, "regexp.Compile") {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
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
		if lit.Value[0] != '"' {
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

		j.Errorf(call, "should use raw string (`...`) with regexp.%s to avoid having to escape twice", sel.Sel.Name)
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintIfReturn(j *lint.Job) {
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
		if !j.IsBoolConst(ret1.Results[0]) {
			return true
		}

		ret2, ok := n2.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		if len(ret2.Results) != 1 {
			return true
		}
		if !j.IsBoolConst(ret2.Results[0]) {
			return true
		}
		j.Errorf(n1, "should use 'return <expr>' instead of 'if <expr> { return <bool> }; return <bool>'")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
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
func (c *Checker) LintRedundantNilCheckWithLen(j *lint.Job) {
	isConstZero := func(expr ast.Expr) (isConst bool, isZero bool) {
		_, ok := expr.(*ast.BasicLit)
		if ok {
			return true, lint.IsZero(expr)
		}
		id, ok := expr.(*ast.Ident)
		if !ok {
			return false, false
		}
		c, ok := j.Program.Info.ObjectOf(id).(*types.Const)
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
		if !j.IsNil(x.Y) {
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
		switch j.Program.Info.TypeOf(xx).(type) {
		case *types.Slice:
			nilType = "nil slices"
		case *types.Map:
			nilType = "nil maps"
		case *types.Chan:
			nilType = "nil channels"
		default:
			return true
		}
		j.Errorf(expr, "should omit nil check; len() for %s is defined as zero", nilType)
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintSlicing(j *lint.Job) {
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
		if _, ok := j.Program.Info.ObjectOf(fun).(*types.Builtin); !ok {
			return true
		}
		arg, ok := call.Args[0].(*ast.Ident)
		if !ok || arg.Obj != s.Obj {
			return true
		}
		j.Errorf(n, "should omit second index in slice, s[a:len(s)] is identical to s[a:]")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
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

func (c *Checker) LintLoopAppend(j *lint.Job) {
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
		if refersTo(j.Program.Info, stmt.Lhs[0], val) {
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
		obj := j.Program.Info.ObjectOf(fun)
		fn, ok := obj.(*types.Builtin)
		if !ok || fn.Name() != "append" {
			return true
		}

		src := j.Program.Info.TypeOf(loop.X)
		dst := j.Program.Info.TypeOf(call.Args[0])
		// TODO(dominikh) remove nil check once Go issue #15173 has
		// been fixed
		if src == nil {
			return true
		}
		if !types.Identical(src, dst) {
			return true
		}

		if j.Render(stmt.Lhs[0]) != j.Render(call.Args[0]) {
			return true
		}

		el, ok := call.Args[1].(*ast.Ident)
		if !ok {
			return true
		}
		if j.Program.Info.ObjectOf(val) != j.Program.Info.ObjectOf(el) {
			return true
		}
		j.Errorf(loop, "should replace loop with %s = append(%s, %s...)",
			j.Render(stmt.Lhs[0]), j.Render(call.Args[0]), j.Render(loop.X))
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintTimeSince(j *lint.Job) {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if !j.IsCallToAST(sel.X, "time.Now") {
			return true
		}
		if sel.Sel.Name != "Sub" {
			return true
		}
		j.Errorf(call, "should use time.Since instead of time.Now().Sub")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintTimeUntil(j *lint.Job) {
	if !j.IsGoVersion(8) {
		return
	}
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !j.IsCallToAST(call, "(time.Time).Sub") {
			return true
		}
		if !j.IsCallToAST(call.Args[0], "time.Now") {
			return true
		}
		j.Errorf(call, "should use time.Until instead of t.Sub(time.Now())")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintSimplerReturn(j *lint.Job) {
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
				if !ok || expr.Op != token.NEQ || !j.IsNil(expr.Y) {
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
					results1 = append(results1, j.Program.Info.ObjectOf(ident))
				}
				if results1[len(results1)-1] != j.Program.Info.ObjectOf(id1) {
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
					results2 = append(results2, j.Program.Info.ObjectOf(ident))
				}
				_, isNil := results2[len(results2)-1].(*types.Nil)
				if results2[len(results2)-1] != j.Program.Info.ObjectOf(id1) &&
					!isNil {
					continue
				}
				for i, v := range results1[:len(results1)-1] {
					if v != results2[i] {
						continue outer
					}
				}

				id1Obj := j.Program.Info.ObjectOf(id1)
				if id1Obj == nil {
					continue
				}
				_, idIface := id1Obj.Type().Underlying().(*types.Interface)
				_, retIface := j.Program.Info.TypeOf(ret.List[len(ret.List)-1].Type).Underlying().(*types.Interface)

				if retIface && !idIface {
					// When the return value is an interface, but the
					// identifier is not, an explicit check for nil is
					// required to return an untyped nil.
					continue
				}

				j.Errorf(ifs, "'if %s != nil { return %s }; return %s' can be simplified to 'return %s'",
					j.Render(expr.X), j.RenderArgs(ret1.Results),
					j.RenderArgs(ret2.Results), j.RenderArgs(ret1.Results))
			}
			return true
		}
		ast.Inspect(node, fn2)
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn1)
	}
}

func (c *Checker) LintReceiveIntoBlank(j *lint.Job) {
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
			j.Errorf(lh, "'_ = <-ch' can be simplified to '<-ch'")
		}
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintFormatInt(j *lint.Job) {
	checkBasic := func(v ast.Expr) bool {
		typ, ok := j.Program.Info.TypeOf(v).(*types.Basic)
		if !ok {
			return false
		}
		return typ.Kind() == types.Int
	}
	checkConst := func(v *ast.Ident) bool {
		c, ok := j.Program.Info.ObjectOf(v).(*types.Const)
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
		basic, ok := j.Program.Info.ObjectOf(v).(*types.Const).Type().(*types.Basic)
		return ok && basic.Kind() == types.UntypedInt
	}

	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !j.IsCallToAST(call, "strconv.FormatInt") {
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
			obj, ok := j.Program.Info.ObjectOf(ident).(*types.TypeName)
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
			j.Errorf(call, "should use strconv.Itoa instead of strconv.FormatInt")
		}
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintSimplerStructConversion(j *lint.Job) {
	fn := func(node ast.Node) bool {
		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		typ1 := j.Program.Info.TypeOf(lit.Type)
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
			typ := j.Program.Info.TypeOf(sel.X)
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
		j.Errorf(node, "should use type conversion instead of struct literal")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintTrim(j *lint.Job) {
	sameNonDynamic := func(node1, node2 ast.Node) bool {
		if reflect.TypeOf(node1) != reflect.TypeOf(node2) {
			return false
		}

		switch node1 := node1.(type) {
		case *ast.Ident:
			return node1.Obj == node2.(*ast.Ident).Obj
		case *ast.SelectorExpr:
			return j.Render(node1) == j.Render(node2)
		case *ast.IndexExpr:
			return j.Render(node1) == j.Render(node2)
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
				n, ok := j.ExprToInt(slice.Low)
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
				s1, ok1 := j.ExprToString(lit)
				s2, ok2 := j.ExprToString(condCall.Args[1])
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
			string, ok1 := j.ExprToString(condCall.Args[1])
			int, ok2 := j.ExprToInt(slice.Low)
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
		j.Errorf(ifstmt, "should replace this if statement with an unconditional %s.%s", pkg, replacement)
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintLoopSlide(j *lint.Job) {
	// TODO(dh): detect bs[i+offset] in addition to bs[offset+i]
	// TODO(dh): consider merging this function with LintLoopCopy
	// TODO(dh): detect length that is an expression, not a variable name
	// TODO(dh): support sliding to a different offset than the beginning of the slice

	fn := func(node ast.Node) bool {
		/*
			for i := 0; i < n; i++ {
				bs[i] = bs[offset+i]
			}

						↓

			copy(bs[:n], bs[offset:offset+n])
		*/

		loop, ok := node.(*ast.ForStmt)
		if !ok || len(loop.Body.List) != 1 || loop.Init == nil || loop.Cond == nil || loop.Post == nil {
			return true
		}
		assign, ok := loop.Init.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 || !lint.IsZero(assign.Rhs[0]) {
			return true
		}
		initvar, ok := assign.Lhs[0].(*ast.Ident)
		if !ok {
			return true
		}
		post, ok := loop.Post.(*ast.IncDecStmt)
		if !ok || post.Tok != token.INC {
			return true
		}
		postvar, ok := post.X.(*ast.Ident)
		if !ok || j.Program.Info.ObjectOf(postvar) != j.Program.Info.ObjectOf(initvar) {
			return true
		}
		bin, ok := loop.Cond.(*ast.BinaryExpr)
		if !ok || bin.Op != token.LSS {
			return true
		}
		binx, ok := bin.X.(*ast.Ident)
		if !ok || j.Program.Info.ObjectOf(binx) != j.Program.Info.ObjectOf(initvar) {
			return true
		}
		biny, ok := bin.Y.(*ast.Ident)
		if !ok {
			return true
		}

		assign, ok = loop.Body.List[0].(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 || assign.Tok != token.ASSIGN {
			return true
		}
		lhs, ok := assign.Lhs[0].(*ast.IndexExpr)
		if !ok {
			return true
		}
		rhs, ok := assign.Rhs[0].(*ast.IndexExpr)
		if !ok {
			return true
		}

		bs1, ok := lhs.X.(*ast.Ident)
		if !ok {
			return true
		}
		bs2, ok := rhs.X.(*ast.Ident)
		if !ok {
			return true
		}
		obj1 := j.Program.Info.ObjectOf(bs1)
		obj2 := j.Program.Info.ObjectOf(bs2)
		if obj1 != obj2 {
			return true
		}
		if _, ok := obj1.Type().Underlying().(*types.Slice); !ok {
			return true
		}

		index1, ok := lhs.Index.(*ast.Ident)
		if !ok || j.Program.Info.ObjectOf(index1) != j.Program.Info.ObjectOf(initvar) {
			return true
		}
		index2, ok := rhs.Index.(*ast.BinaryExpr)
		if !ok || index2.Op != token.ADD {
			return true
		}
		add1, ok := index2.X.(*ast.Ident)
		if !ok {
			return true
		}
		add2, ok := index2.Y.(*ast.Ident)
		if !ok || j.Program.Info.ObjectOf(add2) != j.Program.Info.ObjectOf(initvar) {
			return true
		}

		j.Errorf(loop, "should use copy(%s[:%s], %s[%s:]) instead", j.Render(bs1), j.Render(biny), j.Render(bs1), j.Render(add1))
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintMakeLenCap(j *lint.Job) {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if fn, ok := call.Fun.(*ast.Ident); !ok || fn.Name != "make" {
			// FIXME check whether make is indeed the built-in function
			return true
		}
		switch len(call.Args) {
		case 2:
			// make(T, len)
			if _, ok := j.Program.Info.TypeOf(call.Args[0]).Underlying().(*types.Slice); ok {
				break
			}
			if lint.IsZero(call.Args[1]) {
				j.Errorf(call.Args[1], "should use make(%s) instead", j.Render(call.Args[0]))
			}
		case 3:
			// make(T, len, cap)
			if j.Render(call.Args[1]) == j.Render(call.Args[2]) {
				j.Errorf(call.Args[1], "should use make(%s, %s) instead", j.Render(call.Args[0]), j.Render(call.Args[1]))
			}
		}
		return false
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintAssertNotNil(j *lint.Job) {
	isNilCheck := func(ident *ast.Ident, expr ast.Expr) bool {
		xbinop, ok := expr.(*ast.BinaryExpr)
		if !ok || xbinop.Op != token.NEQ {
			return false
		}
		xident, ok := xbinop.X.(*ast.Ident)
		if !ok || xident.Obj != ident.Obj {
			return false
		}
		if !j.IsNil(xbinop.Y) {
			return false
		}
		return true
	}
	isOKCheck := func(ident *ast.Ident, expr ast.Expr) bool {
		yident, ok := expr.(*ast.Ident)
		if !ok || yident.Obj != ident.Obj {
			return false
		}
		return true
	}
	fn := func(node ast.Node) bool {
		ifstmt, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		assign, ok := ifstmt.Init.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 || !lint.IsBlank(assign.Lhs[0]) {
			return true
		}
		assert, ok := assign.Rhs[0].(*ast.TypeAssertExpr)
		if !ok {
			return true
		}
		binop, ok := ifstmt.Cond.(*ast.BinaryExpr)
		if !ok || binop.Op != token.LAND {
			return true
		}
		assertIdent, ok := assert.X.(*ast.Ident)
		if !ok {
			return true
		}
		assignIdent, ok := assign.Lhs[1].(*ast.Ident)
		if !ok {
			return true
		}
		if !(isNilCheck(assertIdent, binop.X) && isOKCheck(assignIdent, binop.Y)) &&
			!(isNilCheck(assertIdent, binop.Y) && isOKCheck(assignIdent, binop.X)) {
			return true
		}
		j.Errorf(ifstmt, "when %s is true, %s can't be nil", j.Render(assignIdent), j.Render(assertIdent))
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintDeclareAssign(j *lint.Job) {
	fn := func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		if len(block.List) < 2 {
			return true
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

			if refersTo(j.Program.Info, assign.Rhs[0], ident) {
				continue
			}
			j.Errorf(decl, "should merge variable declaration with assignment on next line")
		}
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintBlankOK(j *lint.Job) {
	fn := func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok {
			return true
		}
		if len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
			return true
		}
		if !lint.IsBlank(assign.Lhs[1]) {
			return true
		}
		switch rhs := assign.Rhs[0].(type) {
		case *ast.IndexExpr:
			// The type-checker should make sure that it's a map, but
			// let's be safe.
			if _, ok := j.Program.Info.TypeOf(rhs.X).Underlying().(*types.Map); !ok {
				return true
			}
		case *ast.UnaryExpr:
			if rhs.Op != token.ARROW {
				return true
			}
		default:
			return true
		}
		cp := *assign
		cp.Lhs = cp.Lhs[0:1]
		j.Errorf(assign, "should write %s instead of %s", j.Render(&cp), j.Render(assign))
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintRedundantBreak(j *lint.Job) {
	fn := func(node ast.Node) bool {
		clause, ok := node.(*ast.CaseClause)
		if !ok {
			return true
		}
		if len(clause.Body) < 2 {
			return true
		}
		branch, ok := clause.Body[len(clause.Body)-1].(*ast.BranchStmt)
		if !ok || branch.Tok != token.BREAK || branch.Label != nil {
			return true
		}
		j.Errorf(branch, "redundant break statement")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) Implements(j *lint.Job, typ types.Type, iface string) bool {
	// OPT(dh): we can cache the type lookup
	idx := strings.IndexRune(iface, '.')
	var scope *types.Scope
	var ifaceName string
	if idx == -1 {
		scope = types.Universe
		ifaceName = iface
	} else {
		pkgName := iface[:idx]
		pkg := j.Program.Prog.Package(pkgName)
		if pkg == nil {
			return false
		}
		scope = pkg.Pkg.Scope()
		ifaceName = iface[idx+1:]
	}

	obj := scope.Lookup(ifaceName)
	if obj == nil {
		return false
	}
	i, ok := obj.Type().Underlying().(*types.Interface)
	if !ok {
		return false
	}
	return types.Implements(typ, i)
}

func (c *Checker) LintRedundantSprintf(j *lint.Job) {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !j.IsCallToAST(call, "fmt.Sprintf") {
			return true
		}
		if len(call.Args) != 2 {
			return true
		}
		if s, ok := j.ExprToString(call.Args[0]); !ok || s != "%s" {
			return true
		}
		pkg := j.NodePackage(call)
		arg := call.Args[1]
		typ := pkg.Info.TypeOf(arg)

		if c.Implements(j, typ, "fmt.Stringer") {
			j.Errorf(call, "should use String() instead of fmt.Sprintf")
			return true
		}

		if typ.Underlying() == types.Universe.Lookup("string").Type() {
			if typ == types.Universe.Lookup("string").Type() {
				j.Errorf(call, "the argument is already a string, there's no need to use fmt.Sprintf")
			} else {
				j.Errorf(call, "the argument's underlying type is a string, should use a simple conversion instead of fmt.Sprintf")
			}
		}
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintStringCopy(j *lint.Job) {
	emptyStringLit := func(e ast.Expr) bool {
		bl, ok := e.(*ast.BasicLit)
		return ok && bl.Value == `""`
	}
	fn := func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.BinaryExpr: // "" + s, s + ""
			if x.Op != token.ADD {
				break
			}
			l1 := j.Program.Prog.Fset.Position(x.X.Pos()).Line
			l2 := j.Program.Prog.Fset.Position(x.Y.Pos()).Line
			if l1 != l2 {
				break
			}
			var want ast.Expr
			switch {
			case emptyStringLit(x.X):
				want = x.Y
			case emptyStringLit(x.Y):
				want = x.X
			default:
				return true
			}
			j.Errorf(x, "should use %s instead of %s",
				j.Render(want), j.Render(x))
		case *ast.CallExpr:
			if j.IsCallToAST(x, "fmt.Sprint") && len(x.Args) == 1 {
				// fmt.Sprint(x)

				argT := j.Program.Info.TypeOf(x.Args[0])
				bt, ok := argT.Underlying().(*types.Basic)
				if !ok || bt.Kind() != types.String {
					return true
				}
				if c.Implements(j, argT, "fmt.Stringer") || c.Implements(j, argT, "error") {
					return true
				}

				j.Errorf(x, "should use %s instead of %s", j.Render(x.Args[0]), j.Render(x))
				return true
			}

			// string([]byte(s))
			bt, ok := j.Program.Info.TypeOf(x.Fun).(*types.Basic)
			if !ok || bt.Kind() != types.String {
				break
			}
			nested, ok := x.Args[0].(*ast.CallExpr)
			if !ok {
				break
			}
			st, ok := j.Program.Info.TypeOf(nested.Fun).(*types.Slice)
			if !ok {
				break
			}
			et, ok := st.Elem().(*types.Basic)
			if !ok || et.Kind() != types.Byte {
				break
			}
			xt, ok := j.Program.Info.TypeOf(nested.Args[0]).(*types.Basic)
			if !ok || xt.Kind() != types.String {
				break
			}
			j.Errorf(x, "should use %s instead of %s",
				j.Render(nested.Args[0]), j.Render(x))
		}
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintRedundantReturn(j *lint.Job) {
	fn := func(node ast.Node) bool {
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
			return true
		}
		// if the func has results, a return can't be redundant.
		// similarly, if there are no statements, there can be
		// no return.
		if ret != nil || body == nil || len(body.List) < 1 {
			return true
		}
		rst, ok := body.List[len(body.List)-1].(*ast.ReturnStmt)
		if !ok {
			return true
		}
		// we don't need to check rst.Results as we already
		// checked x.Type.Results to be nil.
		j.Errorf(rst, "redundant return statement")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintErrorsNewSprintf(j *lint.Job) {
	fn := func(node ast.Node) bool {
		if !j.IsCallToAST(node, "errors.New") {
			return true
		}
		call := node.(*ast.CallExpr)
		if !j.IsCallToAST(call.Args[0], "fmt.Sprintf") {
			return true
		}
		j.Errorf(node, "should use fmt.Errorf(...) instead of errors.New(fmt.Sprintf(...))")
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) LintRangeStringRunes(j *lint.Job) {
	sharedcheck.CheckRangeStringRunes(c.nodeFns, j)
}
