package stylecheck // import "honnef.co/go/tools/stylecheck"

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"honnef.co/go/tools/code"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/edit"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/lint"
	. "honnef.co/go/tools/lint/lintdsl"
	"honnef.co/go/tools/pattern"
	"honnef.co/go/tools/report"
	"honnef.co/go/tools/ssa"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

func CheckPackageComment(pass *analysis.Pass) (interface{}, error) {
	// - At least one file in a non-main package should have a package comment
	//
	// - The comment should be of the form
	// "Package x ...". This has a slight potential for false
	// positives, as multiple files can have package comments, in
	// which case they get appended. But that doesn't happen a lot in
	// the real world.

	if pass.Pkg.Name() == "main" {
		return nil, nil
	}
	hasDocs := false
	for _, f := range pass.Files {
		if code.IsInTest(pass, f) {
			continue
		}
		if f.Doc != nil && len(f.Doc.List) > 0 {
			hasDocs = true
			prefix := "Package " + f.Name.Name + " "
			if !strings.HasPrefix(strings.TrimSpace(f.Doc.Text()), prefix) {
				report.Nodef(pass, f.Doc, `package comment should be of the form "%s..."`, prefix)
			}
			f.Doc.Text()
		}
	}

	if !hasDocs {
		for _, f := range pass.Files {
			if code.IsInTest(pass, f) {
				continue
			}
			report.Nodef(pass, f, "at least one file in a package should have a package comment")
		}
	}
	return nil, nil
}

func CheckDotImports(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
	imports:
		for _, imp := range f.Imports {
			path := imp.Path.Value
			path = path[1 : len(path)-1]
			for _, w := range config.For(pass).DotImportWhitelist {
				if w == path {
					continue imports
				}
			}

			if imp.Name != nil && imp.Name.Name == "." && !code.IsInTest(pass, f) {
				report.NodefFG(pass, imp, "should not use dot imports")
			}
		}
	}
	return nil, nil
}

func CheckDuplicatedImports(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		// Collect all imports by their import path
		imports := make(map[string][]*ast.ImportSpec, len(f.Imports))
		for _, imp := range f.Imports {
			imports[imp.Path.Value] = append(imports[imp.Path.Value], imp)
		}

		for path, value := range imports {
			if path[1:len(path)-1] == "unsafe" {
				// Don't flag unsafe. Cgo generated code imports
				// unsafe using the blank identifier, and most
				// user-written cgo code also imports unsafe
				// explicitly.
				continue
			}
			// If there's more than one import per path, we flag that
			if len(value) > 1 {
				s := fmt.Sprintf("duplicate import %s", path)
				for _, imp := range value[1:] {
					pos := lint.DisplayPosition(pass.Fset, imp.Pos())
					s += "\n\t" + "also imported at " + pos.String()
				}
				report.NodefFG(pass, value[0], s)
			}
		}
	}
	return nil, nil
}

func CheckBlankImports(pass *analysis.Pass) (interface{}, error) {
	fset := pass.Fset
	for _, f := range pass.Files {
		if code.IsMainLike(pass) || code.IsInTest(pass, f) {
			continue
		}

		// Collect imports of the form `import _ "foo"`, i.e. with no
		// parentheses, as their comment will be associated with the
		// (paren-free) GenDecl, not the import spec itself.
		//
		// We don't directly process the GenDecl so that we can
		// correctly handle the following:
		//
		//  import _ "foo"
		//  import _ "bar"
		//
		// where only the first import should get flagged.
		skip := map[ast.Spec]bool{}
		ast.Inspect(f, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.File:
				return true
			case *ast.GenDecl:
				if node.Tok != token.IMPORT {
					return false
				}
				if node.Lparen == token.NoPos && node.Doc != nil {
					skip[node.Specs[0]] = true
				}
				return false
			}
			return false
		})
		for i, imp := range f.Imports {
			pos := fset.Position(imp.Pos())

			if !code.IsBlank(imp.Name) {
				continue
			}
			// Only flag the first blank import in a group of imports,
			// or don't flag any of them, if the first one is
			// commented
			if i > 0 {
				prev := f.Imports[i-1]
				prevPos := fset.Position(prev.Pos())
				if pos.Line-1 == prevPos.Line && code.IsBlank(prev.Name) {
					continue
				}
			}

			if imp.Doc == nil && imp.Comment == nil && !skip[imp] {
				report.Nodef(pass, imp, "a blank import should be only in a main or test package, or have a comment justifying it")
			}
		}
	}
	return nil, nil
}

func CheckIncDec(pass *analysis.Pass) (interface{}, error) {
	// TODO(dh): this can be noisy for function bodies that look like this:
	// 	x += 3
	// 	...
	// 	x += 2
	// 	...
	// 	x += 1
	fn := func(node ast.Node) {
		assign := node.(*ast.AssignStmt)
		if assign.Tok != token.ADD_ASSIGN && assign.Tok != token.SUB_ASSIGN {
			return
		}
		if (len(assign.Lhs) != 1 || len(assign.Rhs) != 1) ||
			!code.IsIntLiteral(assign.Rhs[0], "1") {
			return
		}

		suffix := ""
		switch assign.Tok {
		case token.ADD_ASSIGN:
			suffix = "++"
		case token.SUB_ASSIGN:
			suffix = "--"
		}

		report.Nodef(pass, assign, "should replace %s with %s%s", report.Render(pass, assign), report.Render(pass, assign.Lhs[0]), suffix)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.AssignStmt)(nil)}, fn)
	return nil, nil
}

func CheckErrorReturn(pass *analysis.Pass) (interface{}, error) {
fnLoop:
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		sig := fn.Type().(*types.Signature)
		rets := sig.Results()
		if rets == nil || rets.Len() < 2 {
			continue
		}

		if rets.At(rets.Len()-1).Type() == types.Universe.Lookup("error").Type() {
			// Last return type is error. If the function also returns
			// errors in other positions, that's fine.
			continue
		}
		for i := rets.Len() - 2; i >= 0; i-- {
			if rets.At(i).Type() == types.Universe.Lookup("error").Type() {
				pass.Reportf(rets.At(i).Pos(), "error should be returned as the last argument")
				continue fnLoop
			}
		}
	}
	return nil, nil
}

// CheckUnexportedReturn checks that exported functions on exported
// types do not return unexported types.
func CheckUnexportedReturn(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		if fn.Synthetic != "" || fn.Parent() != nil {
			continue
		}
		if !ast.IsExported(fn.Name()) || code.IsMain(pass) || code.IsInTest(pass, fn) {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Recv() != nil && !ast.IsExported(code.Dereference(sig.Recv().Type()).(*types.Named).Obj().Name()) {
			continue
		}
		res := sig.Results()
		for i := 0; i < res.Len(); i++ {
			if named, ok := code.DereferenceR(res.At(i).Type()).(*types.Named); ok &&
				!ast.IsExported(named.Obj().Name()) &&
				named != types.Universe.Lookup("error").Type() {
				pass.Reportf(fn.Pos(), "should not return unexported type")
			}
		}
	}
	return nil, nil
}

func CheckReceiverNames(pass *analysis.Pass) (interface{}, error) {
	ssapkg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).Pkg
	for _, m := range ssapkg.Members {
		if T, ok := m.Object().(*types.TypeName); ok && !T.IsAlias() {
			ms := typeutil.IntuitiveMethodSet(T.Type(), nil)
			for _, sel := range ms {
				fn := sel.Obj().(*types.Func)
				recv := fn.Type().(*types.Signature).Recv()
				if code.Dereference(recv.Type()) != T.Type() {
					// skip embedded methods
					continue
				}
				if recv.Name() == "self" || recv.Name() == "this" {
					report.PosfFG(pass, recv.Pos(), `receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"`)
				}
				if recv.Name() == "_" {
					report.PosfFG(pass, recv.Pos(), "receiver name should not be an underscore, omit the name if it is unused")
				}
			}
		}
	}
	return nil, nil
}

func CheckReceiverNamesIdentical(pass *analysis.Pass) (interface{}, error) {
	ssapkg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).Pkg
	for _, m := range ssapkg.Members {
		names := map[string]int{}

		var firstFn *types.Func
		if T, ok := m.Object().(*types.TypeName); ok && !T.IsAlias() {
			ms := typeutil.IntuitiveMethodSet(T.Type(), nil)
			for _, sel := range ms {
				fn := sel.Obj().(*types.Func)
				recv := fn.Type().(*types.Signature).Recv()
				if code.IsGenerated(pass, recv.Pos()) {
					// Don't concern ourselves with methods in generated code
					continue
				}
				if code.Dereference(recv.Type()) != T.Type() {
					// skip embedded methods
					continue
				}
				if firstFn == nil {
					firstFn = fn
				}
				if recv.Name() != "" && recv.Name() != "_" {
					names[recv.Name()]++
				}
			}
		}

		if len(names) > 1 {
			var seen []string
			for name, count := range names {
				seen = append(seen, fmt.Sprintf("%dx %q", count, name))
			}
			sort.Strings(seen)

			pass.Reportf(firstFn.Pos(), "methods on the same type should have the same receiver name (seen %s)", strings.Join(seen, ", "))
		}
	}
	return nil, nil
}

func CheckContextFirstArg(pass *analysis.Pass) (interface{}, error) {
	// TODO(dh): this check doesn't apply to test helpers. Example from the stdlib:
	// 	func helperCommandContext(t *testing.T, ctx context.Context, s ...string) (cmd *exec.Cmd) {
fnLoop:
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		if fn.Synthetic != "" || fn.Parent() != nil {
			continue
		}
		params := fn.Signature.Params()
		if params.Len() < 2 {
			continue
		}
		if types.TypeString(params.At(0).Type(), nil) == "context.Context" {
			continue
		}
		for i := 1; i < params.Len(); i++ {
			param := params.At(i)
			if types.TypeString(param.Type(), nil) == "context.Context" {
				pass.Reportf(param.Pos(), "context.Context should be the first argument of a function")
				continue fnLoop
			}
		}
	}
	return nil, nil
}

func CheckErrorStrings(pass *analysis.Pass) (interface{}, error) {
	objNames := map[*ssa.Package]map[string]bool{}
	ssapkg := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).Pkg
	objNames[ssapkg] = map[string]bool{}
	for _, m := range ssapkg.Members {
		if typ, ok := m.(*ssa.Type); ok {
			objNames[ssapkg][typ.Name()] = true
		}
	}
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		objNames[fn.Package()][fn.Name()] = true
	}

	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		if code.IsInTest(pass, fn) {
			// We don't care about malformed error messages in tests;
			// they're usually for direct human consumption, not part
			// of an API
			continue
		}
		for _, block := range fn.Blocks {
		instrLoop:
			for _, ins := range block.Instrs {
				call, ok := ins.(*ssa.Call)
				if !ok {
					continue
				}
				if !code.IsCallToAny(call.Common(), "errors.New", "fmt.Errorf") {
					continue
				}

				k, ok := call.Common().Args[0].(*ssa.Const)
				if !ok {
					continue
				}

				s := constant.StringVal(k.Value)
				if len(s) == 0 {
					continue
				}
				switch s[len(s)-1] {
				case '.', ':', '!', '\n':
					pass.Reportf(call.Pos(), "error strings should not end with punctuation or a newline")
				}
				idx := strings.IndexByte(s, ' ')
				if idx == -1 {
					// single word error message, probably not a real
					// error but something used in tests or during
					// debugging
					continue
				}
				word := s[:idx]
				first, n := utf8.DecodeRuneInString(word)
				if !unicode.IsUpper(first) {
					continue
				}
				for _, c := range word[n:] {
					if unicode.IsUpper(c) {
						// Word is probably an initialism or
						// multi-word function name
						continue instrLoop
					}
				}

				word = strings.TrimRightFunc(word, func(r rune) bool { return unicode.IsPunct(r) })
				if objNames[fn.Package()][word] {
					// Word is probably the name of a function or type in this package
					continue
				}
				// First word in error starts with a capital
				// letter, and the word doesn't contain any other
				// capitals, making it unlikely to be an
				// initialism or multi-word function name.
				//
				// It could still be a proper noun, though.

				pass.Reportf(call.Pos(), "error strings should not be capitalized")
			}
		}
	}
	return nil, nil
}

func CheckTimeNames(pass *analysis.Pass) (interface{}, error) {
	suffixes := []string{
		"Sec", "Secs", "Seconds",
		"Msec", "Msecs",
		"Milli", "Millis", "Milliseconds",
		"Usec", "Usecs", "Microseconds",
		"MS", "Ms",
	}
	fn := func(names []*ast.Ident) {
		for _, name := range names {
			if _, ok := pass.TypesInfo.Defs[name]; !ok {
				continue
			}
			T := pass.TypesInfo.TypeOf(name)
			if !code.IsType(T, "time.Duration") && !code.IsType(T, "*time.Duration") {
				continue
			}
			for _, suffix := range suffixes {
				if strings.HasSuffix(name.Name, suffix) {
					report.Nodef(pass, name, "var %s is of type %v; don't use unit-specific suffix %q", name.Name, T, suffix)
					break
				}
			}
		}
	}

	fn2 := func(node ast.Node) {
		switch node := node.(type) {
		case *ast.ValueSpec:
			fn(node.Names)
		case *ast.FieldList:
			for _, field := range node.List {
				fn(field.Names)
			}
		case *ast.AssignStmt:
			if node.Tok != token.DEFINE {
				break
			}
			var names []*ast.Ident
			for _, lhs := range node.Lhs {
				if lhs, ok := lhs.(*ast.Ident); ok {
					names = append(names, lhs)
				}
			}
			fn(names)
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ValueSpec)(nil), (*ast.FieldList)(nil), (*ast.AssignStmt)(nil)}, fn2)
	return nil, nil
}

func CheckErrorVarNames(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				spec := spec.(*ast.ValueSpec)
				if len(spec.Names) != len(spec.Values) {
					continue
				}

				for i, name := range spec.Names {
					val := spec.Values[i]
					if !code.IsCallToAnyAST(pass, val, "errors.New", "fmt.Errorf") {
						continue
					}

					prefix := "err"
					if name.IsExported() {
						prefix = "Err"
					}
					if !strings.HasPrefix(name.Name, prefix) {
						report.Nodef(pass, name, "error var %s should have name of the form %sFoo", name.Name, prefix)
					}
				}
			}
		}
	}
	return nil, nil
}

var httpStatusCodes = map[int]string{
	100: "StatusContinue",
	101: "StatusSwitchingProtocols",
	102: "StatusProcessing",
	200: "StatusOK",
	201: "StatusCreated",
	202: "StatusAccepted",
	203: "StatusNonAuthoritativeInfo",
	204: "StatusNoContent",
	205: "StatusResetContent",
	206: "StatusPartialContent",
	207: "StatusMultiStatus",
	208: "StatusAlreadyReported",
	226: "StatusIMUsed",
	300: "StatusMultipleChoices",
	301: "StatusMovedPermanently",
	302: "StatusFound",
	303: "StatusSeeOther",
	304: "StatusNotModified",
	305: "StatusUseProxy",
	307: "StatusTemporaryRedirect",
	308: "StatusPermanentRedirect",
	400: "StatusBadRequest",
	401: "StatusUnauthorized",
	402: "StatusPaymentRequired",
	403: "StatusForbidden",
	404: "StatusNotFound",
	405: "StatusMethodNotAllowed",
	406: "StatusNotAcceptable",
	407: "StatusProxyAuthRequired",
	408: "StatusRequestTimeout",
	409: "StatusConflict",
	410: "StatusGone",
	411: "StatusLengthRequired",
	412: "StatusPreconditionFailed",
	413: "StatusRequestEntityTooLarge",
	414: "StatusRequestURITooLong",
	415: "StatusUnsupportedMediaType",
	416: "StatusRequestedRangeNotSatisfiable",
	417: "StatusExpectationFailed",
	418: "StatusTeapot",
	422: "StatusUnprocessableEntity",
	423: "StatusLocked",
	424: "StatusFailedDependency",
	426: "StatusUpgradeRequired",
	428: "StatusPreconditionRequired",
	429: "StatusTooManyRequests",
	431: "StatusRequestHeaderFieldsTooLarge",
	451: "StatusUnavailableForLegalReasons",
	500: "StatusInternalServerError",
	501: "StatusNotImplemented",
	502: "StatusBadGateway",
	503: "StatusServiceUnavailable",
	504: "StatusGatewayTimeout",
	505: "StatusHTTPVersionNotSupported",
	506: "StatusVariantAlsoNegotiates",
	507: "StatusInsufficientStorage",
	508: "StatusLoopDetected",
	510: "StatusNotExtended",
	511: "StatusNetworkAuthenticationRequired",
}

func CheckHTTPStatusCodes(pass *analysis.Pass) (interface{}, error) {
	whitelist := map[string]bool{}
	for _, code := range config.For(pass).HTTPStatusCodeWhitelist {
		whitelist[code] = true
	}
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)

		var arg int
		switch code.CallNameAST(pass, call) {
		case "net/http.Error":
			arg = 2
		case "net/http.Redirect":
			arg = 3
		case "net/http.StatusText":
			arg = 0
		case "net/http.RedirectHandler":
			arg = 1
		default:
			return
		}
		lit, ok := call.Args[arg].(*ast.BasicLit)
		if !ok {
			return
		}
		if whitelist[lit.Value] {
			return
		}

		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			return
		}
		s, ok := httpStatusCodes[n]
		if !ok {
			return
		}
		report.NodeFG(pass, lit, fmt.Sprintf("should use constant http.%s instead of numeric literal %d", s, n),
			edit.Fix(fmt.Sprintf("use http.%s instead of %d", s, n), edit.ReplaceWithString(pass.Fset, lit, "http."+s)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckDefaultCaseOrder(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		stmt := node.(*ast.SwitchStmt)
		list := stmt.Body.List
		for i, c := range list {
			if c.(*ast.CaseClause).List == nil && i != 0 && i != len(list)-1 {
				report.NodefFG(pass, c, "default case should be first or last in switch statement")
				break
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.SwitchStmt)(nil)}, fn)
	return nil, nil
}

var (
	checkYodaConditionsQ = pattern.MustParse(`(BinaryExpr left@(BasicLit _ _) tok@(Or "==" "!=") right@(Not (BasicLit _ _)))`)
	checkYodaConditionsR = pattern.MustParse(`(BinaryExpr right tok left)`)
)

func CheckYodaConditions(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if _, edits, ok := MatchAndEdit(pass, checkYodaConditionsQ, checkYodaConditionsR, node); ok {
			report.NodeFG(pass, node, "don't use Yoda conditions",
				edit.Fix("un-Yoda-fy", edits...))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func CheckInvisibleCharacters(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		lit := node.(*ast.BasicLit)
		if lit.Kind != token.STRING {
			return
		}
		for _, r := range lit.Value {
			if unicode.Is(unicode.Cf, r) {
				report.Nodef(pass, lit, "string literal contains the Unicode format character %U, consider using the %q escape sequence", r, r)
			} else if unicode.Is(unicode.Cc, r) && r != '\n' && r != '\t' && r != '\r' {
				report.Nodef(pass, lit, "string literal contains the Unicode control character %U, consider using the %q escape sequence", r, r)
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BasicLit)(nil)}, fn)
	return nil, nil
}

func CheckExportedFunctionDocs(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if code.IsInTest(pass, node) {
			return
		}

		decl := node.(*ast.FuncDecl)
		if decl.Doc == nil {
			return
		}
		if !ast.IsExported(decl.Name.Name) {
			return
		}
		kind := "function"
		if decl.Recv != nil {
			kind = "method"
			switch T := decl.Recv.List[0].Type.(type) {
			case *ast.StarExpr:
				if !ast.IsExported(T.X.(*ast.Ident).Name) {
					return
				}
			case *ast.Ident:
				if !ast.IsExported(T.Name) {
					return
				}
			default:
				ExhaustiveTypeSwitch(T)
			}
		}
		prefix := decl.Name.Name + " "
		if !strings.HasPrefix(decl.Doc.Text(), prefix) {
			report.NodefFG(pass, decl.Doc, `comment on exported %s %s should be of the form "%s..."`, kind, decl.Name.Name, prefix)
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, fn)
	return nil, nil
}

func CheckExportedTypeDocs(pass *analysis.Pass) (interface{}, error) {
	var genDecl *ast.GenDecl
	fn := func(node ast.Node, push bool) bool {
		if !push {
			genDecl = nil
			return false
		}
		if code.IsInTest(pass, node) {
			return false
		}

		switch node := node.(type) {
		case *ast.GenDecl:
			if node.Tok == token.IMPORT {
				return false
			}
			genDecl = node
			return true
		case *ast.TypeSpec:
			if !ast.IsExported(node.Name.Name) {
				return false
			}

			doc := node.Doc
			if doc == nil {
				if len(genDecl.Specs) != 1 {
					// more than one spec in the GenDecl, don't validate the
					// docstring
					return false
				}
				if genDecl.Lparen.IsValid() {
					// 'type ( T )' is weird, don't guess the user's intention
					return false
				}
				doc = genDecl.Doc
				if doc == nil {
					return false
				}
			}

			s := doc.Text()
			articles := [...]string{"A", "An", "The"}
			for _, a := range articles {
				if strings.HasPrefix(s, a+" ") {
					s = s[len(a)+1:]
					break
				}
			}
			if !strings.HasPrefix(s, node.Name.Name+" ") {
				report.NodefFG(pass, doc, `comment on exported type %s should be of the form "%s ..." (with optional leading article)`, node.Name.Name, node.Name.Name)
			}
			return false
		case *ast.FuncLit, *ast.FuncDecl:
			return false
		default:
			ExhaustiveTypeSwitch(node)
			return false
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Nodes([]ast.Node{(*ast.GenDecl)(nil), (*ast.TypeSpec)(nil), (*ast.FuncLit)(nil), (*ast.FuncDecl)(nil)}, fn)
	return nil, nil
}

func CheckExportedVarDocs(pass *analysis.Pass) (interface{}, error) {
	var genDecl *ast.GenDecl
	fn := func(node ast.Node, push bool) bool {
		if !push {
			genDecl = nil
			return false
		}
		if code.IsInTest(pass, node) {
			return false
		}

		switch node := node.(type) {
		case *ast.GenDecl:
			if node.Tok == token.IMPORT {
				return false
			}
			genDecl = node
			return true
		case *ast.ValueSpec:
			if genDecl.Lparen.IsValid() || len(node.Names) > 1 {
				// Don't try to guess the user's intention
				return false
			}
			name := node.Names[0].Name
			if !ast.IsExported(name) {
				return false
			}
			if genDecl.Doc == nil {
				return false
			}
			prefix := name + " "
			if !strings.HasPrefix(genDecl.Doc.Text(), prefix) {
				kind := "var"
				if genDecl.Tok == token.CONST {
					kind = "const"
				}
				report.NodefFG(pass, genDecl.Doc, `comment on exported %s %s should be of the form "%s..."`, kind, name, prefix)
			}
			return false
		case *ast.FuncLit, *ast.FuncDecl:
			return false
		default:
			ExhaustiveTypeSwitch(node)
			return false
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Nodes([]ast.Node{(*ast.GenDecl)(nil), (*ast.ValueSpec)(nil), (*ast.FuncLit)(nil), (*ast.FuncDecl)(nil)}, fn)
	return nil, nil
}
