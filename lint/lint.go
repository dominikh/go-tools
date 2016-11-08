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
	"go/printer"
	"go/token"
	"go/types"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
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

func buildPackage(pkg *types.Package, files []*ast.File, info *types.Info, fset *token.FileSet, mode ssa.BuilderMode) *ssa.Package {
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
	return ssapkg
}

func (l *Linter) Lint(lprog *loader.Program) map[string][]Problem {
	out := map[string][]Problem{}
	for _, pkginfo := range lprog.InitialPackages() {
		ssapkg := buildPackage(pkginfo.Pkg, pkginfo.Files, &pkginfo.Info, lprog.Fset, ssa.GlobalDebug)
		pkg := &Pkg{
			files:     map[string]*File{},
			TypesPkg:  pkginfo.Pkg,
			TypesInfo: pkginfo.Info,
			SSAPkg:    ssapkg,
		}
		for _, file := range pkginfo.Files {
			path := lprog.Fset.Position(file.Pos()).Filename
			f := &File{
				Pkg:      pkg,
				File:     file,
				Filename: path,
				Fset:     lprog.Fset,
			}
			pkg.files[path] = f
			for _, fn := range l.Funcs {
				fn(f)
			}
		}
		sort.Sort(ByPosition(pkg.problems))
		out[pkginfo.Pkg.Path()] = pkg.problems
	}
	return out
}

func (f *File) Source() []byte {
	if f.src != nil {
		return f.src
	}
	path := f.Fset.Position(f.File.Pos()).Filename
	if path != "" {
		f.src, _ = ioutil.ReadFile(path)
	}
	return f.src
}

// pkg represents a package being linted.
type Pkg struct {
	files     map[string]*File
	TypesPkg  *types.Package
	TypesInfo types.Info
	SSAPkg    *ssa.Package

	problems []Problem
}

// file represents a file being linted.
type File struct {
	Pkg      *Pkg
	File     *ast.File
	Filename string
	Fset     *token.FileSet
	src      []byte
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
	return f.Pkg.ErrorfAt(pos, confidence, args...)
}

func (p *Pkg) ErrorfAt(pos token.Position, confidence float64, args ...interface{}) *Problem {
	problem := Problem{
		Position:   pos,
		Confidence: confidence,
	}
	if pos.Filename != "" {
		if f, ok := p.files[pos.Filename]; ok {
			problem.LineText = SrcLine(f.Source(), pos)
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

func (p *Pkg) IsNamedType(typ types.Type, importPath, name string) bool {
	n, ok := typ.(*types.Named)
	if !ok {
		return false
	}
	tn := n.Obj()
	return tn != nil && tn.Pkg() != nil && tn.Pkg().Path() == importPath && tn.Name() == name
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
	line := SrcLine(f.Source(), f.Fset.Position(node.Pos()))
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
	line := SrcLine(f.Source(), f.Fset.Position(node.Pos()))
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

func (f *File) EnclosingSSAFunction(node Positioner) *ssa.Function {
	path, _ := astutil.PathEnclosingInterval(f.File, node.Pos(), node.Pos())
	return ssa.EnclosingFunction(f.Pkg.SSAPkg, path)
}
