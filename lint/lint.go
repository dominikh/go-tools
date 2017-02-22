// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package lint provides the foundation for tools like gosimple.
package lint // import "honnef.co/go/tools/lint"

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
	"honnef.co/go/tools/ssa"
	"honnef.co/go/tools/ssa/ssautil"
)

type Ignore struct {
	Pattern string
	Checks  []string
}

type Program struct {
	SSA      *ssa.Program
	Prog     *loader.Program
	Packages []*Pkg
}

type Func func(*File)

// Problem represents a problem in some source code.
type Problem struct {
	Position token.Position // position in source file
	Text     string         // the prose that describes the problem

	// If the problem has a suggested fix (the minority case),
	// ReplacementLine is a full replacement for the relevant line of the source file.
	ReplacementLine string
}

func (p *Problem) String() string {
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

type Checker interface {
	Init(*Program)
	Funcs() map[string]Func
}

// A Linter lints Go source code.
type Linter struct {
	Checker Checker
	Ignores []Ignore
}

func (l *Linter) ignore(f *File, check string) bool {
	for _, ig := range l.Ignores {
		pkg := f.Pkg.TypesPkg.Path()
		if strings.HasSuffix(pkg, "_test") {
			pkg = pkg[:len(pkg)-len("_test")]
		}
		name := filepath.Join(pkg, filepath.Base(f.Filename))
		if m, _ := filepath.Match(ig.Pattern, name); !m {
			continue
		}
		for _, c := range ig.Checks {
			if m, _ := filepath.Match(c, check); m {
				return true
			}
		}
	}
	return false
}

func (l *Linter) Lint(lprog *loader.Program) map[string][]Problem {
	ssaprog := ssautil.CreateProgram(lprog, ssa.GlobalDebug)
	ssaprog.Build()
	var pkgs []*Pkg
	for _, pkginfo := range lprog.InitialPackages() {
		ssapkg := ssaprog.Package(pkginfo.Pkg)
		pkg := &Pkg{
			TypesPkg:  pkginfo.Pkg,
			TypesInfo: pkginfo.Info,
			SSAPkg:    ssapkg,
			PkgInfo:   pkginfo,
		}
		pkgs = append(pkgs, pkg)
	}
	prog := &Program{
		SSA:      ssaprog,
		Prog:     lprog,
		Packages: pkgs,
	}
	l.Checker.Init(prog)

	funcs := l.Checker.Funcs()
	var keys []string
	for k := range funcs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := map[string][]Problem{}
	type result struct {
		path     string
		problems []Problem
	}
	wg := &sync.WaitGroup{}
	for _, pkg := range pkgs {
		pkg := pkg
		wg.Add(1)
		go func() {
			for _, file := range pkg.PkgInfo.Files {
				path := lprog.Fset.Position(file.Pos()).Filename
				for _, k := range keys {
					f := &File{
						Pkg:      pkg,
						File:     file,
						Filename: path,
						Fset:     lprog.Fset,
						Program:  lprog,
						check:    k,
					}

					fn := funcs[k]
					if fn == nil {
						continue
					}
					if l.ignore(f, k) {
						continue
					}
					fn(f)
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	for _, pkg := range pkgs {
		sort.Sort(ByPosition(pkg.problems))
		out[pkg.PkgInfo.Pkg.Path()] = pkg.problems
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
	TypesPkg  *types.Package
	TypesInfo types.Info
	SSAPkg    *ssa.Package
	PkgInfo   *loader.PackageInfo

	problems []Problem
}

// file represents a file being linted.
type File struct {
	Pkg      *Pkg
	File     *ast.File
	Filename string
	Fset     *token.FileSet
	Program  *loader.Program
	src      []byte
	check    string
}

func (f *File) IsTest() bool { return strings.HasSuffix(f.Filename, "_test.go") }

type Positioner interface {
	Pos() token.Pos
}

func (f *File) Errorf(n Positioner, format string, args ...interface{}) *Problem {
	pos := f.Fset.Position(n.Pos())
	if !pos.IsValid() {
		pos = f.Fset.Position(f.File.Pos())
	}
	return f.Pkg.errorfAt(pos, f.check, format, args...)
}

func (p *Pkg) errorfAt(pos token.Position, check string, format string, args ...interface{}) *Problem {
	problem := Problem{
		Position: pos,
	}

	problem.Text = fmt.Sprintf(format, args...) + fmt.Sprintf(" (%s)", check)
	p.problems = append(p.problems, problem)
	return &p.problems[len(p.problems)-1]
}

func (f *File) IsMain() bool {
	return f.File.Name.Name == "main"
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

func IsZero(expr ast.Expr) bool {
	lit, ok := expr.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}

func (f *File) IsNil(expr ast.Expr) bool {
	return f.Pkg.TypesInfo.Types[expr].IsNil()
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

func (f *File) ExprToInt(expr ast.Expr) (int64, bool) {
	tv := f.Pkg.TypesInfo.Types[expr]
	if tv.Value == nil {
		return 0, false
	}
	if tv.Value.Kind() != constant.Int {
		return 0, false
	}
	return constant.Int64Val(tv.Value)
}

func (f *File) ExprToString(expr ast.Expr) (string, bool) {
	val := f.Pkg.TypesInfo.Types[expr].Value
	if val == nil {
		return "", false
	}
	if val.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(val), true
}

func (f *File) EnclosingSSAFunction(node Positioner) *ssa.Function {
	path, _ := astutil.PathEnclosingInterval(f.File, node.Pos(), node.Pos())
	return ssa.EnclosingFunction(f.Pkg.SSAPkg, path)
}

func (f *File) IsGenerated() bool {
	comments := f.File.Comments
	if len(comments) > 0 {
		comment := comments[0].Text()
		return strings.Contains(comment, "Code generated by") ||
			strings.Contains(comment, "DO NOT EDIT")
	}
	return false
}

func IsGoVersion(version string) bool {
	needle := "go" + version
	// TODO(dh): allow users to pass in a custom build environment
	for _, tag := range build.Default.ReleaseTags {
		if tag == needle {
			return true
		}
	}
	return false
}

func (f *File) IsFunctionCallName(node ast.Node, name string) bool {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	fn, ok := f.Pkg.TypesInfo.ObjectOf(sel.Sel).(*types.Func)
	return ok && fn.FullName() == name
}

func (f *File) IsFunctionCallNameAny(node ast.Node, names ...string) bool {
	for _, name := range names {
		if f.IsFunctionCallName(node, name) {
			return true
		}
	}
	return false
}
