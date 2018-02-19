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

	packages    map[string][2]*Package
	AllPackages []*Package

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
		packages: map[string][2]*Package{},
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
	Importable bool

	bpkg    *bpkg
	checker *types.Checker
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

func (prog *Program) Import(path string, cwd string) (*Package, *Package, error) {
	pkg, xpkg, err := prog.load(path, cwd)
	if err != nil {
		return nil, nil, err
	}
	if err := prog.augment(); err != nil {
		return nil, nil, err
	}
	return pkg, xpkg, nil
}

func (prog *Program) augment() error {
	// When importing a package, we can't immediately type-check its
	// tests, as this may lead to circular dependencies, because
	// unlike the Go compiler, we augment _all_ imported packages with
	// their tests. In order to avoid an infinite loop, we must first
	// type-check all dependencies without their tests, before
	// augmenting them.
	//
	// This also means that we must delay SSA building of all packages
	// until they have been augmented by their tests.
	//
	// The simplest solution is to, upon every user-initiated package
	// import, find all packages that haven't been augmented yet and
	// finish the work. Since this may import even more packages, we
	// have to loop until no more packages are left.
	//
	// OPT(dh): an optimization would maintain an explicit work list,
	// instead of looping over all packages repeatedly, which is
	// quadratic in complexity.
	for augmented := true; augmented; {
		augmented = false
		for _, pkg := range prog.AllPackages {
			// Haven't build SSA form yet
			if pkg.SSA == nil {
				augmented = true
				// package hasn't been augmented with tests yet
				if pkg.bpkg != nil && pkg.checker != nil {
					pkg.Files = append(pkg.Files, pkg.bpkg.testFiles...)
					if err := pkg.checker.Files(pkg.bpkg.testFiles); err != nil {
						return err
					}

					// build XTests package. This has to be done after
					// the main package has been augmented, because
					// external tests get access to identifiers
					// declared in normal tests. SSA form will be
					// built on the next iteration of the outer loop.
					if len(pkg.bpkg.xtestFiles) > 0 {
						bpkg := pkg.bpkg
						pkgPath := bpkg.bp.ImportPath
						xpkg := &Package{
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
							Pkg:   types.NewPackage(pkgPath+"_test", ""),
							Files: bpkg.xtestFiles,
							bpkg:  bpkg,
						}
						prog.typesPackages[xpkg.Pkg] = xpkg
						xpkg.checker = types.NewChecker(prog.Config, prog.Fset, xpkg.Pkg, &xpkg.Info)
						if err := xpkg.checker.Files(bpkg.xtestFiles); err != nil {
							return err
						}
						// Re-register packages, this time with the type-checked xpkg
						prog.packages[bpkg.bp.ImportPath] = [2]*Package{pkg, xpkg}
						// XTests shouldn't be augmented by test files
						xpkg.checker = nil
						xpkg.bpkg = nil
						prog.AllPackages = append(prog.AllPackages, xpkg)
					}

					// Package has been fully augmented now.
					pkg.checker = nil
					pkg.bpkg = nil
				}

				pkg.SSA = prog.SSA.CreatePackage(pkg.Pkg, pkg.Files, &pkg.Info, pkg.Importable)
				for _, f := range pkg.Files {
					tf := prog.Fset.File(f.Pos())
					prog.TokenFileMap[tf] = f
					prog.ASTFileMap[f] = pkg
				}
			}
		}
	}
	prog.SSA.Build()
	return nil
}

func (prog *Program) CreateFromFiles(path string, files ...*ast.File) (*Package, error) {
	// prefetch build.Packages of dependencies
	for _, f := range files {
		for _, imp := range f.Imports {
			go prog.findPackage(imp.Path.Value, "")
		}
	}

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
	prog.AllPackages = append(prog.AllPackages, pkg)

	if err := prog.augment(); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (prog *Program) createFromBpkg(bpkg *bpkg) (*Package, *Package, error) {
	if c, ok := prog.packages[bpkg.bp.ImportPath]; ok {
		return c[0], c[1], nil
	}

	// prefetch build.Packages of dependencies
	for _, imp := range bpkg.bp.Imports {
		go prog.findPackage(imp, bpkg.bp.Dir)
	}
	for _, imp := range bpkg.bp.TestImports {
		go prog.findPackage(imp, bpkg.bp.Dir)
	}
	// for _, imp := range bpkg.bp.XTestImports {
	// 	go prog.findPackage(imp, bpkg.bp.Dir)
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
		Bpkg:       bpkg.bp,
		Pkg:        types.NewPackage(pkgPath, ""),
		Files:      make([]*ast.File, 0, len(bpkg.files)+len(bpkg.testFiles)),
		bpkg:       bpkg,
		Importable: true,
	}
	pkg.Files = append(pkg.Files, bpkg.files...)
	prog.typesPackages[pkg.Pkg] = pkg
	pkg.checker = types.NewChecker(prog.Config, prog.Fset, pkg.Pkg, &pkg.Info)
	if err := pkg.checker.Files(bpkg.files); err != nil {
		return nil, nil, err
	}
	prog.packages[bpkg.bp.ImportPath] = [2]*Package{pkg, nil}

	prog.AllPackages = append(prog.AllPackages, pkg)

	return pkg, nil, nil
}

func (prog *Program) load(path string, cwd string) (*Package, *Package, error) {
	if path == "unsafe" {
		return prog.unsafe, nil, nil
	}

	bpkg, err := prog.findPackage(path, cwd)
	if err != nil {
		return nil, nil, err
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
	pkg, _, err := imp.prog.load(path, dir)
	if err != nil {
		return nil, err
	}
	return pkg.Pkg, nil
}
