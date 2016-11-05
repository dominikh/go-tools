// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package lint provides the foundation for tools like gosimple.
package lint // import "honnef.co/go/lint"

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/gcimporter15"
	"golang.org/x/tools/go/ssa"
)

type Func func(*File)

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

type ByPosition []Problem

func (p ByPosition) Len() int      { return len(p) }
func (p ByPosition) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p ByPosition) Less(i, j int) bool {
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

// A Linter lints Go source code.
type Linter struct {
	Funcs []Func
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
	pkg := &Pkg{
		fset:  token.NewFileSet(),
		files: make(map[string]*File),
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
		pkg.files[filename] = &File{
			Pkg:      pkg,
			File:     f,
			Fset:     pkg.fset,
			Src:      src,
			Filename: filename,
		}
	}
	return pkg.lint(l.Funcs), nil
}

// pkg represents a package being linted.
type Pkg struct {
	fset  *token.FileSet
	files map[string]*File

	TypesPkg  *types.Package
	TypesInfo *types.Info
	SSAPkg    *ssa.Package

	// main is whether this is a "main" package.
	main bool

	problems []Problem
}

func (p *Pkg) lint(linters []Func) []Problem {
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

	p.main = p.IsMain()

	for _, f := range p.files {
		p.lintFile(f, linters)
	}

	sort.Sort(ByPosition(p.problems))

	return p.problems
}

func (p *Pkg) lintFile(f *File, linters []Func) {
	for _, fn := range linters {
		fn(f)
	}
}

// file represents a file being linted.
type File struct {
	Pkg      *Pkg
	File     *ast.File
	Fset     *token.FileSet
	Src      []byte
	Filename string
}

func (f *File) IsTest() bool { return strings.HasSuffix(f.Filename, "_test.go") }

type Link string
type Category string

type Positioner interface {
	Pos() token.Pos
}

// The variadic arguments may start with link and category types,
// and must end with a format string and any arguments.
// It returns the new Problem.
func (f *File) Errorf(n Positioner, confidence float64, args ...interface{}) *Problem {
	pos := f.Fset.Position(n.Pos())
	if pos.Filename == "" {
		pos.Filename = f.Filename
	}
	return f.Pkg.ErrorfAt(pos, confidence, args...)
}

func (p *Pkg) ErrorfAt(pos token.Position, confidence float64, args ...interface{}) *Problem {
	problem := Problem{
		Position:   pos,
		Confidence: confidence,
	}
	if pos.Filename != "" {
		// The file might not exist in our mapping if a //line directive was encountered.
		if f, ok := p.files[pos.Filename]; ok {
			problem.LineText = SrcLine(f.Src, pos)
		}
	}

argLoop:
	for len(args) > 1 { // always leave at least the format string in args
		switch v := args[0].(type) {
		case Link:
			problem.Link = string(v)
		case Category:
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

// BuildPackage builds an SSA program with IR for a single package.
//
// It populates pkg by type-checking the specified file ASTs.  All
// dependencies are loaded using the importer specified by tc, which
// typically loads compiler export data; SSA code cannot be built for
// those packages.  BuildPackage then constructs an ssa.Program with all
// dependency packages created, and builds and returns the SSA package
// corresponding to pkg.
//
// The caller must have set pkg.Path() to the import path.
//
// The operation fails if there were any type-checking or import errors.
//
// See ../ssa/example_test.go for an example.
//
func buildPackage(tc *types.Config, fset *token.FileSet, pkg *types.Package, files []*ast.File, mode ssa.BuilderMode) (*ssa.Package, *types.Info, error) {
	if fset == nil {
		panic("no token.FileSet")
	}
	if pkg.Path() == "" {
		panic("package has no import path")
	}

	info := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Scopes:     make(map[ast.Node]*types.Scope),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	if err := types.NewChecker(tc, fset, pkg, info).Files(files); err != nil {
		return nil, info, err
	}

	prog := ssa.NewProgram(fset, mode)

	// Create SSA packages for all imports.
	// Order is not significant.
	created := make(map[*types.Package]bool)
	var createAll func(pkgs []*types.Package)
	createAll = func(pkgs []*types.Package) {
		for _, p := range pkgs {
			if !created[p] {
				created[p] = true
				prog.CreatePackage(p, nil, nil, true)
				createAll(p.Imports())
			}
		}
	}
	createAll(pkg.Imports())

	// Create and build the primary package.
	ssapkg := prog.CreatePackage(pkg, files, info, false)
	ssapkg.Build()
	return ssapkg, info, nil
}

func (p *Pkg) typeCheck() error {
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
	var anyFile *File
	var astFiles []*ast.File
	for _, f := range p.files {
		anyFile = f
		astFiles = append(astFiles, f.File)
	}
	pkg := types.NewPackage(anyFile.File.Name.Name, "")
	ssapkg, info, err := buildPackage(config, p.fset, pkg, astFiles, ssa.GlobalDebug)
	// Remember the typechecking info, even if config.Check failed,
	// since we will get partial information.
	p.TypesPkg = pkg
	p.TypesInfo = info
	if err != nil {
		return err
	}
	p.SSAPkg = ssapkg
	return nil
}

func (p *Pkg) IsNamedType(typ types.Type, importPath, name string) bool {
	n, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	tn := n.Obj()
	return tn != nil && tn.Pkg() != nil && tn.Pkg().Path() == importPath && tn.Name() == name
}

// scopeOf returns the tightest scope encompassing id.
func (p *Pkg) ScopeOf(id *ast.Ident) *types.Scope {
	var scope *types.Scope
	if obj := p.TypesInfo.ObjectOf(id); obj != nil {
		scope = obj.Parent()
	}
	if scope == p.TypesPkg.Scope() {
		// We were given a top-level identifier.
		// Use the file-level scope instead of the package-level scope.
		pos := id.Pos()
		for _, f := range p.files {
			if f.File.Pos() <= pos && pos < f.File.End() {
				scope = p.TypesInfo.Scopes[f.File]
				break
			}
		}
	}
	return scope
}

func (p *Pkg) IsMain() bool {
	for _, f := range p.files {
		if f.IsMain() {
			return true
		}
	}
	return false
}

func (f *File) IsMain() bool {
	return f.File.Name.Name == "main"
}

// exportedType reports whether typ is an exported type.
// It is imprecise, and will err on the side of returning true,
// such as for composite types.
func ExportedType(typ types.Type) bool {
	switch T := typ.(type) {
	case *types.Named:
		// Builtin types have no package.
		return T.Obj().Pkg() == nil || T.Obj().Exported()
	case *types.Map:
		return ExportedType(T.Key()) && ExportedType(T.Elem())
	case interface {
		Elem() types.Type
	}: // array, slice, pointer, chan
		return ExportedType(T.Elem())
	}
	// Be conservative about other types, such as struct, interface, etc.
	return true
}

func ReceiverType(fn *ast.FuncDecl) string {
	switch e := fn.Recv.List[0].Type.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return e.X.(*ast.Ident).Name
	}
	panic(fmt.Sprintf("unknown method receiver AST node type %T", fn.Recv.List[0].Type))
}

func (f *File) Walk(fn func(ast.Node) bool) {
	ast.Inspect(f.File, fn)
}

func (f *File) Render(x interface{}) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, f.Fset, x); err != nil {
		panic(err)
	}
	return buf.String()
}

func (f *File) debugRender(x interface{}) string {
	var buf bytes.Buffer
	if err := ast.Fprint(&buf, f.Fset, x, nil); err != nil {
		panic(err)
	}
	return buf.String()
}

func (f *File) RenderArgs(args []ast.Expr) string {
	var ss []string
	for _, arg := range args {
		ss = append(ss, f.Render(arg))
	}
	return strings.Join(ss, ", ")
}

func IsIdent(expr ast.Expr, ident string) bool {
	id, ok := expr.(*ast.Ident)
	return ok && id.Name == ident
}

// isBlank returns whether id is the blank identifier "_".
// If id == nil, the answer is false.
func IsBlank(id ast.Expr) bool {
	ident, ok := id.(*ast.Ident)
	return ok && ident.Name == "_"
}

func IsPkgDot(expr ast.Expr, pkg, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	return ok && IsIdent(sel.X, pkg) && IsIdent(sel.Sel, name)
}

func IsZero(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}

func IsOne(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "1"
}

func IsNil(expr ast.Expr) bool {
	// FIXME(dominikh): use type information
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
func (f *File) IsUntypedConst(expr ast.Expr) (defType string, ok bool) {
	// Re-evaluate expr outside of its context to see if it's untyped.
	// (An expr evaluated within, for example, an assignment context will get the type of the LHS.)
	exprStr := f.Render(expr)
	tv, err := types.Eval(f.Fset, f.Pkg.TypesPkg, expr.Pos(), exprStr)
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
func (f *File) FirstLineOf(node, match ast.Node) string {
	line := f.Render(node)
	if i := strings.Index(line, "\n"); i >= 0 {
		line = line[:i]
	}
	return f.IndentOf(match) + line
}

func (f *File) IndentOf(node ast.Node) string {
	line := SrcLine(f.Src, f.Fset.Position(node.Pos()))
	for i, r := range line {
		switch r {
		case ' ', '\t':
		default:
			return line[:i]
		}
	}
	return line // unusual or empty line
}

func (f *File) SrcLineWithMatch(node ast.Node, pattern string) (m []string) {
	line := SrcLine(f.Src, f.Fset.Position(node.Pos()))
	line = strings.TrimSuffix(line, "\n")
	rx := regexp.MustCompile(pattern)
	return rx.FindStringSubmatch(line)
}

// srcLine returns the complete line at p, including the terminating newline.
func SrcLine(src []byte, p token.Position) string {
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

func (f *File) BoolConst(expr ast.Expr) bool {
	val := f.Pkg.TypesInfo.ObjectOf(expr.(*ast.Ident)).(*types.Const).Val()
	return constant.BoolVal(val)
}

func (f *File) IsBoolConst(expr ast.Expr) bool {
	// We explicitly don't support typed bools because more often than
	// not, custom bool types are used as binary enums and the
	// explicit comparison is desired.

	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	obj := f.Pkg.TypesInfo.ObjectOf(ident)
	c, ok := obj.(*types.Const)
	if !ok {
		return false
	}
	basic, ok := c.Type().(*types.Basic)
	if !ok {
		return false
	}
	if basic.Kind() != types.UntypedBool && basic.Kind() != types.Bool {
		return false
	}
	return true
}

func ExprToInt(expr ast.Expr) (string, bool) {
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
