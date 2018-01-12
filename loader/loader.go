package loader

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"honnef.co/go/tools/ssa"
)

type Statistics struct {
	Finding      time.Duration
	Parsing      time.Duration
	TypeChecking time.Duration
	SSA          time.Duration
}

func (stat Statistics) String() string {
	return fmt.Sprintf("finding = %s, parsing = %s, type checking = %s, SSA = %s",
		stat.Finding, stat.Parsing, stat.TypeChecking, stat.SSA)
}

type Program struct {
	Fset   *token.FileSet
	Build  *build.Context
	Config *types.Config
	SSA    *ssa.Program

	Packages   map[string]*Package
	Statistics Statistics

	unsafe *Package

	bpkgsMu sync.Mutex
	bpkgs   map[string]*bpkg
}

type Package struct {
	Pkg   *types.Package
	Bpkg  *build.Package
	SSA   *ssa.Package
	Files []*ast.File
	types.Info
}

func (prog *Program) Init() {
	prog.Config.Importer = importer{prog}
	prog.Packages = map[string]*Package{}
	prog.unsafe = &Package{
		Pkg: types.Unsafe,
		SSA: prog.SSA.CreatePackage(types.Unsafe, nil, nil, true),
	}
	prog.bpkgs = map[string]*bpkg{}
}

func (prog *Program) parsePackage(bpkg *build.Package) ([]*ast.File, error) {
	files := make([]*ast.File, len(bpkg.GoFiles))
	for i, name := range bpkg.GoFiles {
		path := filepath.Join(bpkg.Dir, name)
		f, err := parser.ParseFile(prog.Fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files[i] = f
	}
	return files, nil
}

type bpkg struct {
	bp    *build.Package
	files []*ast.File
	err   error
	ready chan struct{} // closed to broadcast readiness
}

func (prog *Program) findPackage(path, dir string) (*build.Package, []*ast.File, error) {
	bp, err := prog.Build.Import(path, dir, build.FindOnly)
	if err != nil {
		return bp, nil, err
	}
	prog.bpkgsMu.Lock()
	v, ok := prog.bpkgs[bp.ImportPath]
	if ok {
		prog.bpkgsMu.Unlock()
		<-v.ready
	} else {
		v = &bpkg{ready: make(chan struct{})}
		prog.bpkgs[bp.ImportPath] = v
		prog.bpkgsMu.Unlock()

		v.bp, v.err = prog.Build.Import(path, dir, 0)
		if v.err == nil {
			t := time.Now()
			v.files, v.err = prog.parsePackage(v.bp)
			d := time.Since(t)
			atomic.AddInt64((*int64)(&prog.Statistics.Parsing), int64(d))
		}
		close(v.ready)
	}
	return v.bp, v.files, v.err
}

func (prog *Program) Import(path string, cwd string) (*Package, error) {
	pkg, err := prog.load(path, cwd)
	if err != nil {
		return nil, err
	}
	t := time.Now()
	prog.SSA.Build()
	prog.Statistics.SSA += time.Since(t)
	return pkg, nil
}

func (prog *Program) load(path string, cwd string) (*Package, error) {
	if path == "unsafe" {
		return prog.unsafe, nil
	}

	pkg := Package{
		Info: types.Info{
			Types:      map[ast.Expr]types.TypeAndValue{},
			Defs:       map[*ast.Ident]types.Object{},
			Uses:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Scopes:     map[ast.Node]*types.Scope{},
			InitOrder:  []*types.Initializer{},
		},
	}
	var err error
	t := time.Now()
	pkg.Bpkg, pkg.Files, err = prog.findPackage(path, cwd)
	if err != nil {
		return nil, err
	}

	if c, ok := prog.Packages[pkg.Bpkg.ImportPath]; ok {
		return c, nil
	}
	prog.Statistics.Finding += time.Since(t)

	for _, imp := range pkg.Bpkg.Imports {
		// prefetch build.Packages of dependencies
		go prog.findPackage(imp, pkg.Bpkg.Dir)
	}

	pkg.Pkg, err = prog.Config.Check(pkg.Bpkg.ImportPath, prog.Fset, pkg.Files, &pkg.Info)
	if err != nil {
		return nil, err
	}

	t = time.Now()
	pkg.SSA = prog.SSA.CreatePackage(pkg.Pkg, pkg.Files, &pkg.Info, true)
	prog.Statistics.SSA += time.Since(t)

	prog.Packages[pkg.Bpkg.ImportPath] = &pkg
	return &pkg, nil
}

type importer struct {
	prog *Program
}

func (imp importer) Import(path string) (*types.Package, error) {
	return imp.ImportFrom(path, "", 0)
}

func (imp importer) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	pkg, err := imp.prog.load(path, dir)
	if err != nil {
		return nil, err
	}
	return pkg.Pkg, nil
}
