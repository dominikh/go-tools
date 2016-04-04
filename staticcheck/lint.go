// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package simple contains a linter for Go source code.
package staticcheck // import "honnef.co/go/staticcheck"

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	htmltemplate "html/template"
	"regexp"
	"sort"
	"strconv"
	"strings"
	texttemplate "text/template"
	"time"

	"golang.org/x/tools/go/gcimporter15"
)

const styleGuideBase = "https://golang.org/wiki/CodeReviewComments"

// A Linter lints Go source code.
type Linter struct {
}

// Problem represents a problem in some source code.
type Problem struct {
	Position   token.Position // position in source file
	Text       string         // the prose that describes the problem
	Link       string         // (optional) the link to the style guide for the problem
	Confidence float64        // a value in (0,1] estimating the confidence in this problem's correctness
	LineText   string         // the source line
	Category   string         // a short name for the general category of the problem

	// If the problem has a suggested fix (the minority case),
	// ReplacementLine is a full replacement for the relevant line of the source file.
	ReplacementLine string
}

func (p *Problem) String() string {
	if p.Link != "" {
		return p.Text + "\n\n" + p.Link
	}
	return p.Text
}

type byPosition []Problem

func (p byPosition) Len() int      { return len(p) }
func (p byPosition) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p byPosition) Less(i, j int) bool {
	pi, pj := p[i].Position, p[j].Position

	if pi.Filename != pj.Filename {
		return pi.Filename < pj.Filename
	}
	if pi.Line != pj.Line {
		return pi.Line < pj.Line
	}
	if pi.Column != pj.Column {
		return pi.Column < pj.Column
	}

	return p[i].Text < p[j].Text
}

// Lint lints src.
func (l *Linter) Lint(filename string, src []byte) ([]Problem, error) {
	return l.LintFiles(map[string][]byte{filename: src})
}

// LintFiles lints a set of files of a single package.
// The argument is a map of filename to source.
func (l *Linter) LintFiles(files map[string][]byte) ([]Problem, error) {
	if len(files) == 0 {
		return nil, nil
	}
	pkg := &pkg{
		fset:  token.NewFileSet(),
		files: make(map[string]*file),
	}
	var pkgName string
	for filename, src := range files {
		f, err := parser.ParseFile(pkg.fset, filename, src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		if pkgName == "" {
			pkgName = f.Name.Name
		} else if f.Name.Name != pkgName {
			return nil, fmt.Errorf("%s is in package %s, not %s", filename, f.Name.Name, pkgName)
		}
		pkg.files[filename] = &file{
			pkg:      pkg,
			f:        f,
			fset:     pkg.fset,
			src:      src,
			filename: filename,
		}
	}
	return pkg.lint(), nil
}

// pkg represents a package being linted.
type pkg struct {
	fset  *token.FileSet
	files map[string]*file

	typesPkg  *types.Package
	typesInfo *types.Info

	// sortable is the set of types in the package that implement sort.Interface.
	sortable map[string]bool
	// main is whether this is a "main" package.
	main bool

	problems []Problem
}

func (p *pkg) lint() []Problem {
	if err := p.typeCheck(); err != nil {
		/* TODO(dsymonds): Consider reporting these errors when golint operates on entire packages.
		if e, ok := err.(types.Error); ok {
			pos := p.fset.Position(e.Pos)
			conf := 1.0
			if strings.Contains(e.Msg, "can't find import: ") {
				// Golint is probably being run in a context that doesn't support
				// typechecking (e.g. package files aren't found), so don't warn about it.
				conf = 0
			}
			if conf > 0 {
				p.errorfAt(pos, conf, category("typechecking"), e.Msg)
			}

			// TODO(dsymonds): Abort if !e.Soft?
		}
		*/
	}

	p.scanSortable()
	p.main = p.isMain()

	for _, f := range p.files {
		f.lint()
	}

	sort.Sort(byPosition(p.problems))

	return p.problems
}

// file represents a file being linted.
type file struct {
	pkg      *pkg
	f        *ast.File
	fset     *token.FileSet
	src      []byte
	filename string
}

func (f *file) isTest() bool { return strings.HasSuffix(f.filename, "_test.go") }

func (f *file) lint() {
	f.checkRegexps()
	f.checkTemplate()
	f.checkTimeParse()
	f.checkEncodingBinary()
	f.checkTimeSleepConstant()
}

type link string
type category string

// The variadic arguments may start with link and category types,
// and must end with a format string and any arguments.
// It returns the new Problem.
func (f *file) errorf(n ast.Node, confidence float64, args ...interface{}) *Problem {
	pos := f.fset.Position(n.Pos())
	if pos.Filename == "" {
		pos.Filename = f.filename
	}
	return f.pkg.errorfAt(pos, confidence, args...)
}

func (p *pkg) errorfAt(pos token.Position, confidence float64, args ...interface{}) *Problem {
	problem := Problem{
		Position:   pos,
		Confidence: confidence,
	}
	if pos.Filename != "" {
		// The file might not exist in our mapping if a //line directive was encountered.
		if f, ok := p.files[pos.Filename]; ok {
			problem.LineText = srcLine(f.src, pos)
		}
	}

argLoop:
	for len(args) > 1 { // always leave at least the format string in args
		switch v := args[0].(type) {
		case link:
			problem.Link = string(v)
		case category:
			problem.Category = string(v)
		default:
			break argLoop
		}
		args = args[1:]
	}

	problem.Text = fmt.Sprintf(args[0].(string), args[1:]...)

	p.problems = append(p.problems, problem)
	return &p.problems[len(p.problems)-1]
}

var gcImporter = gcimporter.Import

// importer implements go/types.Importer.
// It also implements go/types.ImporterFrom, which was new in Go 1.6,
// so vendoring will work.
type importer struct {
	impFn    func(packages map[string]*types.Package, path, srcDir string) (*types.Package, error)
	packages map[string]*types.Package
}

func (i importer) Import(path string) (*types.Package, error) {
	return i.impFn(i.packages, path, "")
}

// (importer).ImportFrom is in lint16.go.

func (p *pkg) typeCheck() error {
	config := &types.Config{
		// By setting a no-op error reporter, the type checker does as much work as possible.
		Error: func(error) {},
		Importer: importer{
			impFn:    gcImporter,
			packages: make(map[string]*types.Package),
		},
	}
	info := &types.Info{
		Types:  make(map[ast.Expr]types.TypeAndValue),
		Defs:   make(map[*ast.Ident]types.Object),
		Uses:   make(map[*ast.Ident]types.Object),
		Scopes: make(map[ast.Node]*types.Scope),
	}
	var anyFile *file
	var astFiles []*ast.File
	for _, f := range p.files {
		anyFile = f
		astFiles = append(astFiles, f.f)
	}
	pkg, err := config.Check(anyFile.f.Name.Name, p.fset, astFiles, info)
	// Remember the typechecking info, even if config.Check failed,
	// since we will get partial information.
	p.typesPkg = pkg
	p.typesInfo = info
	return err
}

func (p *pkg) typeOf(expr ast.Expr) types.Type {
	if p.typesInfo == nil {
		return nil
	}
	return p.typesInfo.TypeOf(expr)
}

func (p *pkg) isNamedType(typ types.Type, importPath, name string) bool {
	n, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	tn := n.Obj()
	return tn != nil && tn.Pkg() != nil && tn.Pkg().Path() == importPath && tn.Name() == name
}

// scopeOf returns the tightest scope encompassing id.
func (p *pkg) scopeOf(id *ast.Ident) *types.Scope {
	var scope *types.Scope
	if obj := p.typesInfo.ObjectOf(id); obj != nil {
		scope = obj.Parent()
	}
	if scope == p.typesPkg.Scope() {
		// We were given a top-level identifier.
		// Use the file-level scope instead of the package-level scope.
		pos := id.Pos()
		for _, f := range p.files {
			if f.f.Pos() <= pos && pos < f.f.End() {
				scope = p.typesInfo.Scopes[f.f]
				break
			}
		}
	}
	return scope
}

func (p *pkg) scanSortable() {
	p.sortable = make(map[string]bool)

	// bitfield for which methods exist on each type.
	const (
		Len = 1 << iota
		Less
		Swap
	)
	nmap := map[string]int{"Len": Len, "Less": Less, "Swap": Swap}
	has := make(map[string]int)
	for _, f := range p.files {
		f.walk(func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
				return true
			}
			// TODO(dsymonds): We could check the signature to be more precise.
			recv := receiverType(fn)
			if i, ok := nmap[fn.Name.Name]; ok {
				has[recv] |= i
			}
			return false
		})
	}
	for typ, ms := range has {
		if ms == Len|Less|Swap {
			p.sortable[typ] = true
		}
	}
}

func (p *pkg) isMain() bool {
	for _, f := range p.files {
		if f.isMain() {
			return true
		}
	}
	return false
}

func (f *file) isMain() bool {
	if f.f.Name.Name == "main" {
		return true
	}
	return false
}

// exportedType reports whether typ is an exported type.
// It is imprecise, and will err on the side of returning true,
// such as for composite types.
func exportedType(typ types.Type) bool {
	switch T := typ.(type) {
	case *types.Named:
		// Builtin types have no package.
		return T.Obj().Pkg() == nil || T.Obj().Exported()
	case *types.Map:
		return exportedType(T.Key()) && exportedType(T.Elem())
	case interface {
		Elem() types.Type
	}: // array, slice, pointer, chan
		return exportedType(T.Elem())
	}
	// Be conservative about other types, such as struct, interface, etc.
	return true
}

func receiverType(fn *ast.FuncDecl) string {
	switch e := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return e.X.(*ast.Ident).Name
	}
	panic(fmt.Sprintf("unknown method receiver AST node type %T", fn.Recv.List[0].Type))
}

func (f *file) walk(fn func(ast.Node) bool) {
	ast.Inspect(f.f, fn)
}

func (f *file) render(x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, f.fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}

func (f *file) debugRender(x interface{}) string {
	var buf bytes.Buffer
	if err := ast.Fprint(&buf, f.fset, x, nil); err != nil {
		panic(err)
	}
	return buf.String()
}

func (f *file) renderArgs(args []ast.Expr) string {
	var ss []string
	for _, arg := range args {
		ss = append(ss, f.render(arg))
	}
	return strings.Join(ss, ", ")
}

func isIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}

// isBlank returns whether id is the blank identifier "_".
// If id == nil, the answer is false.
func isBlank(id ast.Expr) bool {
	ident, ok := id.(*ast.Ident)
	return ok && ident.Name == "_"
}

func isPkgDot(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, pkg) && isIdent(sel.Sel, name)
}

func isZero(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}

func isOne(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "1"
}

func isNil(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == "nil"
}

var basicTypeKinds = map[types.BasicKind]string{
	types.UntypedBool:    "bool",
	types.UntypedInt:     "int",
	types.UntypedRune:    "rune",
	types.UntypedFloat:   "float64",
	types.UntypedComplex: "complex128",
	types.UntypedString:  "string",
}

// isUntypedConst reports whether expr is an untyped constant,
// and indicates what its default type is.
// scope may be nil.
func (f *file) isUntypedConst(expr ast.Expr) (defType string, ok bool) {
	// Re-evaluate expr outside of its context to see if it's untyped.
	// (An expr evaluated within, for example, an assignment context will get the type of the LHS.)
	exprStr := f.render(expr)
	tv, err := types.Eval(f.fset, f.pkg.typesPkg, expr.Pos(), exprStr)
	if err != nil {
		return "", false
	}
	if b, ok := tv.Type.(*types.Basic); ok {
		if dt, ok := basicTypeKinds[b.Kind()]; ok {
			return dt, true
		}
	}

	return "", false
}

// firstLineOf renders the given node and returns its first line.
// It will also match the indentation of another node.
func (f *file) firstLineOf(node, match ast.Node) string {
	line := f.render(node)
	if i := strings.Index(line, "\n"); i >= 0 {
		line = line[:i]
	}
	return f.indentOf(match) + line
}

func (f *file) indentOf(node ast.Node) string {
	line := srcLine(f.src, f.fset.Position(node.Pos()))
	for i, r := range line {
		switch r {
		case ' ', '\t':
		default:
			return line[:i]
		}
	}
	return line // unusual or empty line
}

func (f *file) srcLineWithMatch(node ast.Node, pattern string) (m []string) {
	line := srcLine(f.src, f.fset.Position(node.Pos()))
	line = strings.TrimSuffix(line, "\n")
	rx := regexp.MustCompile(pattern)
	return rx.FindStringSubmatch(line)
}

// srcLine returns the complete line at p, including the terminating newline.
func srcLine(src []byte, p token.Position) string {
	// Run to end of line in both directions if not at line start/end.
	lo, hi := p.Offset, p.Offset+1
	for lo > 0 && src[lo-1] != '\n' {
		lo--
	}
	for hi < len(src) && src[hi-1] != '\n' {
		hi++
	}
	return string(src[lo:hi])
}

func exprToInt(expr ast.Expr) (string, bool) {
	switch y := expr.(type) {
	case *ast.BasicLit:
		if y.Kind != token.INT {
			return "", false
		}
		return y.Value, true
	case *ast.UnaryExpr:
		if y.Op != token.SUB && y.Op != token.ADD {
			return "", false
		}
		x, ok := y.X.(*ast.BasicLit)
		if !ok {
			return "", false
		}
		if x.Kind != token.INT {
			return "", false
		}
		v := constant.MakeFromLiteral(x.Value, x.Kind, 0)
		return constant.UnaryOp(y.Op, v, 0).String(), true
	default:
		return "", false
	}
}

func (f *file) checkRegexps() {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isPkgDot(call.Fun, "regexp", "MustCompile") &&
			!isPkgDot(call.Fun, "regexp", "Compile") {
			return true
		}
		if len(call.Args) != 1 {
			return true
		}
		typ := f.pkg.typesInfo.Types[call.Args[0]]
		if typ.Value == nil {
			return true
		}
		if typ.Value.Kind() != constant.String {
			return true
		}
		s := constant.StringVal(typ.Value)
		_, err := regexp.Compile(s)
		if err != nil {
			f.errorf(call.Args[0], 1, category("FIXME"), "%s", err)
		}
		return true
	}
	f.walk(fn)
}

func (f *file) checkTemplate() {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if len(call.Args) != 1 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name != "Parse" {
			return true
		}
		var kind string
		typ := f.pkg.typesInfo.TypeOf(sel.X)
		if typ == nil {
			return true
		}
		switch typ.String() {
		case "*text/template.Template":
			kind = "text"
		case "*html/template.Template":
			kind = "html"
		default:
			return true
		}

		val := f.pkg.typesInfo.Types[call.Args[0]].Value
		if val == nil {
			return true
		}
		if val.Kind() != constant.String {
			return true
		}
		s := constant.StringVal(val)
		var err error
		switch kind {
		case "text":
			_, err = texttemplate.New("").Parse(s)
		case "html":
			_, err = htmltemplate.New("").Parse(s)
		}
		if err != nil {
			// TODO(dominikh): whitelist other parse errors, if any
			if strings.Contains(err.Error(), "unexpected") {
				f.errorf(call.Args[0], 1, category("FIXME"), "%s", err)
			}
		}
		return true
	}
	f.walk(fn)
}

func (f *file) checkTimeParse() {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isPkgDot(call.Fun, "time", "Parse") {
			return true
		}
		if len(call.Args) != 2 {
			return true
		}
		typ := f.pkg.typesInfo.Types[call.Args[0]]
		if typ.Value == nil {
			return true
		}
		if typ.Value.Kind() != constant.String {
			return true
		}
		s := constant.StringVal(typ.Value)
		s = strings.Replace(s, "_", " ", -1)
		s = strings.Replace(s, "Z", "-", -1)
		_, err := time.Parse(s, s)
		if err != nil {
			f.errorf(call.Args[0], 1, category("FIXME"), "%s", err)
		}
		return true
	}
	f.walk(fn)
}

func (f *file) checkEncodingBinary() {
	// TODO(dominikh): also check binary.Read
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isPkgDot(call.Fun, "binary", "Write") {
			return true
		}
		if len(call.Args) != 3 {
			return true
		}
		dataType := f.pkg.typesInfo.TypeOf(call.Args[2]).Underlying()
		if typ, ok := dataType.(*types.Pointer); ok {
			dataType = typ.Elem().Underlying()
		}
		if typ, ok := dataType.(interface {
			Elem() types.Type
		}); ok {
			if _, ok := typ.(*types.Pointer); !ok {
				dataType = typ.Elem()
			}
		}

		if validEncodingBinaryType(dataType) {
			return true
		}
		f.errorf(call.Args[2], 1, category("FIXME"), "type %s cannot be used with binary.Write",
			f.pkg.typesInfo.TypeOf(call.Args[2]))
		return true
	}
	f.walk(fn)
}

func validEncodingBinaryType(typ types.Type) bool {
	typ = typ.Underlying()
	switch typ := typ.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Uint8, types.Uint16, types.Uint32, types.Uint64,
			types.Int8, types.Int16, types.Int32, types.Int64,
			types.Float32, types.Float64, types.Complex64, types.Complex128, types.Invalid:
			return true
		}
		return false
	case *types.Struct:
		n := typ.NumFields()
		for i := 0; i < n; i++ {
			if !validEncodingBinaryType(typ.Field(i).Type()) {
				return false
			}
		}
		return true
	case *types.Array:
		return validEncodingBinaryType(typ.Elem())
	case *types.Interface:
		// we can't determine if it's a valid type or not
		return true
	}
	return false
}

func (f *file) checkTimeSleepConstant() {
	fn := func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if !isPkgDot(call.Fun, "time", "Sleep") {
			return true
		}
		if len(call.Args) != 1 {
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			return true
		}
		if n == 0 || n > 120 {
			// time.Sleep(0) is a seldomly used pattern in concurrency
			// tests. >120 might be intentional. 120 was chosen
			// because the user could've meant 2 minutes.
			return true
		}
		recommendation := "time.Sleep(time.Nanosecond)"
		if n != 1 {
			recommendation = fmt.Sprintf("time.Sleep(%d * time.Nanosecond)", n)
		}
		f.errorf(call.Args[0], 1, category("FIXME"), "sleeping for %d nanoseconds is probably a bug. Be explicit if it isn't: %s", n, recommendation)
		return true
	}
	f.walk(fn)
}
