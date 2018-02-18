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

	"honnef.co/go/tools/ssa"
)

// XXX make Import safe for concurrent use

type Program struct {
	Fset   *token.FileSet
	Build  *build.Context
	Config *types.Config
	SSA    *ssa.Program

	Packages map[string]*Package

	unsafe *Package

	bpkgsMu sync.Mutex
	bpkgs   map[string]*bpkg

	TokenFileMap map[*token.File]*ast.File
	ASTFileMap   map[*ast.File]*Package

	typesPackages map[*types.Package]*Package
}

func (prog *Program) PackageFromTypesPackage(tpkg *types.Package) *Package {
	return prog.typesPackages[tpkg]
}

func NewProgram() *Program {
	fset := token.NewFileSet()
	ssaprog := ssa.NewProgram(fset, ssa.GlobalDebug)
	b := build.Default
	b.CgoEnabled = false
	prog := &Program{
		Fset:     fset,
		Build:    &b,
		Config:   &types.Config{},
		SSA:      ssaprog,
		Packages: map[string]*Package{},
		unsafe: &Package{
			Pkg: types.Unsafe,
			SSA: ssaprog.CreatePackage(types.Unsafe, nil, nil, true),
		},
		bpkgs:         map[string]*bpkg{},
		TokenFileMap:  map[*token.File]*ast.File{},
		ASTFileMap:    map[*ast.File]*Package{},
		typesPackages: map[*types.Package]*Package{},
	}
	prog.Config.Importer = importer{prog}
	return prog
}

type Package struct {
	Pkg   *types.Package
	Bpkg  *build.Package
	SSA   *ssa.Package
	Files []*ast.File
	types.Info

	augmented bool
}

func (pkg *Package) String() string {
	return fmt.Sprintf("package %s // import %q", pkg.Pkg.Name(), pkg.Pkg.Path())
}

const (
	goFiles = iota
	testFiles
	xtestFiles
)

func (prog *Program) parsePackage(bpkg *build.Package, which int) ([]*ast.File, error) {
	var in []string
	switch which {
	case goFiles:
		in = bpkg.GoFiles
	case testFiles:
		in = bpkg.TestGoFiles
	case xtestFiles:
		in = bpkg.XTestGoFiles
	default:
		panic(fmt.Sprintf("invalid value for which: %d", which))
	}
	files := make([]*ast.File, len(in))
	for i, name := range in {
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
	bp         *build.Package
	files      []*ast.File
	testFiles  []*ast.File
	xtestFiles []*ast.File
	err        error
	ready      chan struct{} // closed to broadcast readiness
}

func (prog *Program) findPackage(path, dir string) (*bpkg, error) {
	bp, err := prog.Build.Import(path, dir, build.FindOnly)
	if err != nil {
		return nil, err
	}
	prog.bpkgsMu.Lock()
	v, ok := prog.bpkgs[bp.ImportPath]
	if ok {
		prog.bpkgsMu.Unlock()
		<-v.ready
	} else {
		v = &bpkg{ready: make(chan struct{})}
		defer close(v.ready)
		prog.bpkgs[bp.ImportPath] = v
		prog.bpkgsMu.Unlock()

		v.bp, v.err = prog.Build.Import(path, dir, 0)
		if v.err == nil {
			v.files, v.err = prog.parsePackage(v.bp, goFiles)
			if v.err != nil {
				return v, v.err
			}
			v.testFiles, v.err = prog.parsePackage(v.bp, testFiles)
			if v.err != nil {
				return v, v.err
			}
			v.xtestFiles, v.err = prog.parsePackage(v.bp, xtestFiles)
			if v.err != nil {
				return v, v.err
			}
		}
	}
	return v, v.err
}

func (prog *Program) Import(path string, cwd string, tests bool) (*Package, error) {
	pkg, err := prog.load(path, cwd)
	if err != nil {
		return nil, err
	}
	if tests && !pkg.augmented {
		// XXX augment import with tests
		//
		// XXX what to do about SSA? if the package was already
		// imported without tests, SSA building will have already
		// finished.
		//
		// XXX also need to add test files to pkg.Files
	}
	prog.SSA.Build()
	return pkg, nil
}

func (prog *Program) CreateFromFiles(path string, files ...*ast.File) (*Package, error) {
	// prefetch build.Packages of dependencies
	for _, f := range files {
		for _, imp := range f.Imports {
			go prog.findPackage(imp.Path.Value, "")
		}
	}
	// if tests {
	// 	for _, imp := range bpkg.bp.TestImports {
	// 		go prog.findPackage(imp, bpkg.bp.Dir)
	// 	}
	// 	for _, imp := range bpkg.bp.XTestImports {
	// 		go prog.findPackage(imp, bpkg.bp.Dir)
	// 	}
	// }

	pkgPath := path
	pkg := &Package{
		Info: types.Info{
			Types:      map[ast.Expr]types.TypeAndValue{},
			Defs:       map[*ast.Ident]types.Object{},
			Uses:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Scopes:     map[ast.Node]*types.Scope{},
			InitOrder:  []*types.Initializer{},
		},
		Bpkg:  nil,
		Pkg:   types.NewPackage(pkgPath, ""),
		Files: files,
	}
	prog.typesPackages[pkg.Pkg] = pkg

	c := types.NewChecker(prog.Config, prog.Fset, pkg.Pkg, &pkg.Info)
	if err := c.Files(files); err != nil {
		return nil, err
	}
	pkg.SSA = prog.SSA.CreatePackage(pkg.Pkg, pkg.Files, &pkg.Info, true)
	prog.SSA.Build()
	// prog.Packages[bpkg.bp.ImportPath] = pkg

	for _, f := range pkg.Files {
		tf := prog.Fset.File(f.Pos())
		prog.TokenFileMap[tf] = f
		prog.ASTFileMap[f] = pkg
	}
	return pkg, nil
}

func (prog *Program) createFromBpkg(bpkg *bpkg) (*Package, error) {
	if c, ok := prog.Packages[bpkg.bp.ImportPath]; ok {
		return c, nil
	}

	// prefetch build.Packages of dependencies
	for _, imp := range bpkg.bp.Imports {
		go prog.findPackage(imp, bpkg.bp.Dir)
	}
	// if tests {
	// 	for _, imp := range bpkg.bp.TestImports {
	// 		go prog.findPackage(imp, bpkg.bp.Dir)
	// 	}
	// 	for _, imp := range bpkg.bp.XTestImports {
	// 		go prog.findPackage(imp, bpkg.bp.Dir)
	// 	}
	// }

	pkgPath := bpkg.bp.ImportPath
	pkg := &Package{
		Info: types.Info{
			Types:      map[ast.Expr]types.TypeAndValue{},
			Defs:       map[*ast.Ident]types.Object{},
			Uses:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Scopes:     map[ast.Node]*types.Scope{},
			InitOrder:  []*types.Initializer{},
		},
		Bpkg:  bpkg.bp,
		Pkg:   types.NewPackage(pkgPath, ""),
		Files: bpkg.files,
	}
	prog.typesPackages[pkg.Pkg] = pkg

	c := types.NewChecker(prog.Config, prog.Fset, pkg.Pkg, &pkg.Info)
	if err := c.Files(bpkg.files); err != nil {
		return nil, err
	}
	pkg.SSA = prog.SSA.CreatePackage(pkg.Pkg, pkg.Files, &pkg.Info, true)
	prog.Packages[bpkg.bp.ImportPath] = pkg

	for _, f := range pkg.Files {
		tf := prog.Fset.File(f.Pos())
		prog.TokenFileMap[tf] = f
		prog.ASTFileMap[f] = pkg
	}
	return pkg, nil
}

func (prog *Program) load(path string, cwd string) (*Package, error) {
	if path == "unsafe" {
		return prog.unsafe, nil
	}

	bpkg, err := prog.findPackage(path, cwd)
	if err != nil {
		return nil, err
	}
	return prog.createFromBpkg(bpkg)
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
