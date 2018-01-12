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
}

func (prog *Program) parsePackage(pkg *Package) error {
	wg := sync.WaitGroup{}
	pkg.Files = make([]*ast.File, len(pkg.Bpkg.GoFiles))
	wg.Add(len(pkg.Bpkg.GoFiles))
	errch := make(chan error, 1)
	for i, name := range pkg.Bpkg.GoFiles {
		go func(i int, name string) {
			path := filepath.Join(pkg.Bpkg.Dir, name)
			f, err := parser.ParseFile(prog.Fset, path, nil, parser.ParseComments)
			if err != nil {
				select {
				case errch <- err:
				default:
				}
			}
			pkg.Files[i] = f
			wg.Done()
		}(i, name)
	}
	wg.Wait()
	select {
	case err := <-errch:
		return err
	default:
	}
	return nil
}

func (prog *Program) Import(path string, cwd string) (*Package, error) {
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
	pkg.Bpkg, err = prog.Build.Import(path, cwd, 0)
	if err != nil {
		return nil, err
	}

	if c, ok := prog.Packages[pkg.Bpkg.ImportPath]; ok {
		return c, nil
	}
	prog.Statistics.Finding += time.Since(t)

	t = time.Now()
	if err := prog.parsePackage(&pkg); err != nil {
		return nil, err
	}
	prog.Statistics.Parsing += time.Since(t)

	pkg.Pkg, err = prog.Config.Check(pkg.Bpkg.ImportPath, prog.Fset, pkg.Files, &pkg.Info)
	if err != nil {
		return nil, err
	}

	t = time.Now()
	pkg.SSA = prog.SSA.CreatePackage(pkg.Pkg, pkg.Files, &pkg.Info, true)
	pkg.SSA.Build()
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
	pkg, err := imp.prog.Import(path, dir)
	if err != nil {
		return nil, err
	}
	return pkg.Pkg, nil
}
