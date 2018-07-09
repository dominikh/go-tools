package loader

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"path/filepath"
	"sync"
)

type errChan chan error

func newErrChan() errChan { return make(errChan, 1) }

func (ch errChan) send(err error) {
	select {
	case ch <- err:
	default:
	}
}

func (ch errChan) recv() error {
	select {
	case err := <-ch:
		return err
	default:
		return nil
	}
}

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

	if len(in) == 0 {
		if which != goFiles || len(bpkg.CgoFiles) == 0 {
			return nil, nil
		}
	}

	files := make([]*ast.File, len(in))
	errCh := newErrChan()
	wg := sync.WaitGroup{}
	wg.Add(len(in))
	for i, name := range in {
		i, name := i, name
		go func() {
			defer wg.Done()
			path := filepath.Join(bpkg.Dir, name)
			f, err := parser.ParseFile(prog.Fset, path, nil, parser.ParseComments)
			if err != nil {
				errCh.send(err)
				return
			}
			files[i] = f
		}()
	}
	wg.Wait()
	if err := errCh.recv(); err != nil {
		return nil, err
	}

	if which == goFiles && bpkg.CgoFiles != nil {
		cgoFiles, err := processCgoFiles(bpkg, prog.Fset, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, cgoFiles...)
	}

	return files, nil
}

func (prog *Program) importBuildPackage(path, srcDir string) (lpkg *BuildPackage, cached bool, err error) {
	bpkg, err := prog.Build.Import(path, srcDir, build.FindOnly)
	if err != nil {
		return nil, false, err
	}

	prog.bpkgsMu.Lock()
	lpkg, cached = prog.buildPackages[bpkg.ImportPath]
	if cached {
		prog.bpkgsMu.Unlock()
		<-lpkg.ready
	} else {
		lpkg = &BuildPackage{
			ready: make(chan struct{}),
		}
		defer close(lpkg.ready)
		prog.buildPackages[bpkg.ImportPath] = lpkg
		prog.bpkgsMu.Unlock()

		var root *build.Package
		root, lpkg.err = prog.Build.Import(path, srcDir, 0)
		if lpkg.err != nil {
			return nil, false, lpkg.err
		}
		lpkg.Bpkg = root

		lpkg.GoFiles, lpkg.err = prog.parsePackage(root, goFiles)
		if lpkg.err != nil {
			return nil, false, lpkg.err
		}

		lpkg.TestFiles, lpkg.err = prog.parsePackage(root, testFiles)
		if lpkg.err != nil {
			return nil, false, lpkg.err
		}

		lpkg.XTestFiles, lpkg.err = prog.parsePackage(root, xtestFiles)
		if lpkg.err != nil {
			return nil, false, lpkg.err
		}

	}
	return lpkg, cached, lpkg.err
}

func (prog *Program) importBuildPackageTree(path, srcDir string, stack []string) (*BuildPackage, error) {
	lpkg, cached, err := prog.importBuildPackage(path, srcDir)
	if err != nil {
		return nil, err
	}
	for i, s := range stack {
		if s == lpkg.Bpkg.ImportPath {
			s := ""
			s += fmt.Sprintln("package", stack[i])
			for _, s := range stack[i+1:] {
				s += fmt.Sprintln("\timports", s)
			}
			s += fmt.Sprintln("\timports", path)
			return nil, errors.New("import loop:\n" + s)
		}
	}
	// Returning early isn't just an optimization, but also necessary
	// to break import cycles caused by tests.
	if cached {
		return lpkg, lpkg.err
	}

	root := lpkg.Bpkg
	stack = append(stack, root.ImportPath)

	wg := sync.WaitGroup{}
	wg.Add(len(root.Imports))
	wg.Add(len(root.XTestImports))
	wg.Add(len(root.TestImports))

	errCh := newErrChan()
	for _, imp := range root.Imports {
		if imp == "C" {
			wg.Done()
			continue
		}
		go func(imp string) {
			_, err := prog.importBuildPackageTree(imp, root.Dir, stack[0:len(stack):len(stack)])
			if err != nil {
				errCh.send(err)
			}
			wg.Done()
		}(imp)
	}
	for _, imp := range root.TestImports {
		if imp == "C" {
			wg.Done()
			continue
		}
		go func(imp string) {
			// Only pass the last element of the stack. The tests
			// of the dependencies of our tests are free to import us,
			// as such an import won't depend on our tests.
			//
			// In other words, tests form their own root. In yet other
			// words, tests break dependency chains.
			_, err := prog.importBuildPackageTree(imp, root.Dir, stack[len(stack)-1:len(stack):len(stack)])
			if err != nil {
				errCh.send(err)
			}
			wg.Done()
		}(imp)
	}
	for _, imp := range root.XTestImports {
		if imp == "C" {
			wg.Done()
			continue
		}
		go func(imp string) {
			// nil stack because XTest packages constitute their own,
			// independent, unimportable package.
			_, err := prog.importBuildPackageTree(imp, root.Dir, nil)
			if err != nil {
				errCh.send(err)
			}
			wg.Done()
		}(imp)
	}

	wg.Wait()

	if err := errCh.recv(); err != nil {
		lpkg.err = err
		return nil, err
	}

	if len(root.CgoFiles) != 0 && root.ImportPath != "runtime/cgo" {
		// For CgoFiles, we only process the imports that the user
		// provided. Cgo preprocessing, however, adds its own imports
		// that we have to handle specially.
		_, err := prog.importBuildPackageTree("runtime/cgo", root.Dir, stack[0:len(stack):len(stack)])
		if err != nil {
			lpkg.err = err
			return nil, err
		}
	}

	return lpkg, lpkg.err
}
