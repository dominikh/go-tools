package loader

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"sync"

	"honnef.co/go/tools/ssa"
)

// XXX make Import safe for concurrent use

type Program struct {
	Fset   *token.FileSet
	Build  *build.Context
	Config *types.Config
	SSA    *ssa.Program

	pkgsMu      sync.Mutex
	packages    map[string][2]*Package
	AllPackages []*Package

	unsafe *Package

	bpkgsMu       sync.Mutex
	buildPackages map[string]*BuildPackage

	TokenFileMap map[*token.File]*ast.File
	ASTFileMap   map[*ast.File]*Package

	typesPackages map[*types.Package]*Package
}

func (prog *Program) PackageFromTypesPackage(tpkg *types.Package) *Package {
	return prog.typesPackages[tpkg]
}

func NewProgram(ctx *build.Context) *Program {
	fset := token.NewFileSet()
	ssaprog := ssa.NewProgram(fset, ssa.GlobalDebug)
	prog := &Program{
		Fset:  fset,
		Build: ctx,
		Config: &types.Config{
			Sizes: types.SizesFor(ctx.Compiler, ctx.GOARCH),
		},
		SSA:      ssaprog,
		packages: map[string][2]*Package{},
		unsafe: &Package{
			Pkg: types.Unsafe,
			SSA: ssaprog.CreatePackage(types.Unsafe, nil, nil, true),
		},
		buildPackages: map[string]*BuildPackage{},
		TokenFileMap:  map[*token.File]*ast.File{},
		ASTFileMap:    map[*ast.File]*Package{},
		typesPackages: map[*types.Package]*Package{},
	}
	prog.Config.Importer = importer{prog}
	return prog
}

type BuildPackage struct {
	Bpkg       *build.Package
	GoFiles    []*ast.File
	TestFiles  []*ast.File
	XTestFiles []*ast.File

	ready chan struct{}
	err   error
}

type Package struct {
	Pkg   *types.Package
	Bpkg  *BuildPackage
	SSA   *ssa.Package
	Files []*ast.File
	types.Info
	Importable bool

	checker *types.Checker
	ready   chan struct{}
	err     error
}

func (pkg *Package) String() string {
	return fmt.Sprintf("package %s // import %q", pkg.Pkg.Name(), pkg.Pkg.Path())
}

const (
	goFiles = iota
	testFiles
	xtestFiles
)

func (prog *Program) Import(path string, cwd string) (*Package, *Package, error) {
	bpkg, err := prog.importBuildPackageTree(path, cwd, nil)
	if err != nil {
		return nil, nil, err
	}

	wg := sync.WaitGroup{}
	wg.Add(len(prog.buildPackages))
	for _, bp := range prog.buildPackages {
		go func(bp *BuildPackage) {
			prog.load(bp)
			wg.Done()
		}(bp)
	}
	wg.Wait()
	if err := prog.augment(); err != nil {
		return nil, nil, err
	}
	return prog.load(bpkg)
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
				if pkg.checker != nil {
					pkg.Files = append(pkg.Files, pkg.Bpkg.TestFiles...)
					if err := pkg.checker.Files(pkg.Bpkg.TestFiles); err != nil {
						return err
					}

					// build XTests package. This has to be done after
					// the main package has been augmented, because
					// external tests get access to identifiers
					// declared in normal tests. SSA form will be
					// built on the next iteration of the outer loop.
					if len(pkg.Bpkg.XTestFiles) > 0 {
						bpkg := pkg.Bpkg
						pkgPath := bpkg.Bpkg.ImportPath
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
							Bpkg:  bpkg,
							Pkg:   types.NewPackage(pkgPath+"_test", ""),
							Files: bpkg.XTestFiles,
						}
						prog.typesPackages[xpkg.Pkg] = xpkg
						xpkg.checker = types.NewChecker(prog.Config, prog.Fset, xpkg.Pkg, &xpkg.Info)
						if err := xpkg.checker.Files(bpkg.XTestFiles); err != nil {
							return err
						}
						// Re-register packages, this time with the type-checked xpkg
						prog.packages[bpkg.Bpkg.ImportPath] = [2]*Package{pkg, xpkg}
						// XTests shouldn't be augmented by test files
						xpkg.checker = nil
						prog.AllPackages = append(prog.AllPackages, xpkg)
					}

					// Package has been fully augmented now.
					pkg.checker = nil
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
			v := imp.Path.Value
			v = v[1 : len(v)-1]
			_, err := prog.importBuildPackageTree(v, "", nil)
			if err != nil {
				return nil, err
			}
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

func (prog *Program) createFromBpkg(bpkg *BuildPackage) (*Package, *Package, error) {
	prog.pkgsMu.Lock()
	pkgs, ok := prog.packages[bpkg.Bpkg.ImportPath]
	if ok {
		prog.pkgsMu.Unlock()
		<-pkgs[0].ready
	} else {
		pkgs[0] = &Package{
			Info: types.Info{
				Types:      map[ast.Expr]types.TypeAndValue{},
				Defs:       map[*ast.Ident]types.Object{},
				Uses:       map[*ast.Ident]types.Object{},
				Implicits:  map[ast.Node]types.Object{},
				Selections: map[*ast.SelectorExpr]*types.Selection{},
				Scopes:     map[ast.Node]*types.Scope{},
				InitOrder:  []*types.Initializer{},
			},
			Files:      make([]*ast.File, 0, len(bpkg.GoFiles)+len(bpkg.TestFiles)),
			Bpkg:       bpkg,
			Pkg:        types.NewPackage(bpkg.Bpkg.ImportPath, ""),
			Importable: true,
			ready:      make(chan struct{}),
		}
		defer close(pkgs[0].ready)
		prog.packages[bpkg.Bpkg.ImportPath] = pkgs
		prog.AllPackages = append(prog.AllPackages, pkgs[0])
		prog.typesPackages[pkgs[0].Pkg] = pkgs[0]
		prog.pkgsMu.Unlock()

		pkgs[0].Files = append(pkgs[0].Files, bpkg.GoFiles...)
		pkgs[0].checker = types.NewChecker(prog.Config, prog.Fset, pkgs[0].Pkg, &pkgs[0].Info)
		if pkgs[0].err = pkgs[0].checker.Files(bpkg.GoFiles); pkgs[0].err != nil {
			return nil, nil, pkgs[0].err
		}
	}
	return pkgs[0], pkgs[1], pkgs[0].err
}

func (prog *Program) load(pkg *BuildPackage) (*Package, *Package, error) {
	if pkg.Bpkg.ImportPath == "unsafe" {
		return prog.unsafe, nil, nil
	}

	return prog.createFromBpkg(pkg)
}

type importer struct {
	prog *Program
}

func (imp importer) Import(path string) (*types.Package, error) {
	return imp.ImportFrom(path, "", 0)
}

func (imp importer) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	bpkg, cached, err := imp.prog.importBuildPackage(path, dir)
	if err != nil {
		return nil, err
	}
	if !cached {
		panic(fmt.Sprintf("internal error: BuildPackage for (%q, %q) wasn't loaded yet", path, dir))
	}
	pkg, _, err := imp.prog.load(bpkg)
	if err != nil {
		return nil, err
	}
	return pkg.Pkg, nil
}
