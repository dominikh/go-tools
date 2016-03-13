package main

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/kisielk/gotool"
)

var exitCode int

func main() {
	// FIXME check flag.NArgs
	paths := gotool.ImportPaths([]string{os.Args[1]})
	cwd, err := os.Getwd()
	if err != nil {
		// XXX
		log.Fatal(err)
	}
	for _, path := range paths {
		pkg, err := build.Import(path, cwd, build.FindOnly)
		if err != nil {
			// XXX
			log.Fatal(err)
		}
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, pkg.Dir, nil, 0)
		if err != nil {
			// XXX
			log.Fatal(err)
		}
		for _, pkg := range pkgs {
			doPackage(fset, pkg)
		}

	}
	os.Exit(exitCode)
}

type Package struct {
	p    *ast.Package
	fset *token.FileSet
	decl map[string]ast.Node
	used map[string]bool
}

func doPackage(fset *token.FileSet, pkg *ast.Package) {
	p := &Package{
		p:    pkg,
		fset: fset,
		decl: make(map[string]ast.Node),
		used: make(map[string]bool),
	}
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			switch n := decl.(type) {
			case *ast.GenDecl:
				// var, const, types
				for _, spec := range n.Specs {
					switch s := spec.(type) {
					case *ast.ValueSpec:
						// constants and variables.
						for _, name := range s.Names {
							p.decl[name.Name] = n
						}
					case *ast.TypeSpec:
						// type definitions.
						p.decl[s.Name.Name] = n
					}
				}
			case *ast.FuncDecl:
				// function declarations
				// TODO(remy): do methods
				if n.Recv == nil {
					p.decl[n.Name.Name] = n
				}
			}
		}
	}
	// init() and _ are always used
	p.used["init"] = true
	p.used["_"] = true
	if pkg.Name != "main" {
		// exported names are marked used for non-main packages.
		for name := range p.decl {
			if ast.IsExported(name) {
				p.used[name] = true
			}
		}
	} else {
		// in main programs, main() is called.
		p.used["main"] = true
	}
	for _, file := range pkg.Files {
		// walk file looking for used nodes.
		ast.Walk(p, file)
	}
	// reports.
	var reports Reports
	for name, node := range p.decl {
		if !p.used[name] {
			pos := node.Pos()
			if node, ok := node.(*ast.GenDecl); ok && node.Lparen.IsValid() {
				for _, spec := range node.Specs {
					switch spec := spec.(type) {
					case *ast.ValueSpec:
						for _, s := range spec.Names {
							if s.Name == name {
								pos = s.NamePos
								break
							}
						}
					case *ast.TypeSpec:
						pos = spec.Name.Pos()
					}
				}
			}
			reports = append(reports, Report{pos, name})
		}
	}
	sort.Sort(reports)
	for _, report := range reports {
		fmt.Printf("%s: %s is unused\n", fset.Position(report.pos), report.name)
	}
}

type Report struct {
	pos  token.Pos
	name string
}
type Reports []Report

func (l Reports) Len() int           { return len(l) }
func (l Reports) Less(i, j int) bool { return l[i].pos < l[j].pos }
func (l Reports) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// Visits files for used nodes.
func (p *Package) Visit(node ast.Node) ast.Visitor {
	u := usedWalker(*p) // hopefully p fields are references.
	switch n := node.(type) {
	// don't walk whole file, but only:
	case *ast.ValueSpec:
		// - variable initializers
		for _, value := range n.Values {
			ast.Walk(&u, value)
		}
		// variable types.
		if n.Type != nil {
			ast.Walk(&u, n.Type)
		}
	case *ast.BlockStmt:
		// - function bodies
		for _, stmt := range n.List {
			ast.Walk(&u, stmt)
		}
	case *ast.FuncDecl:
		// - function signatures
		ast.Walk(&u, n.Type)
	case *ast.TypeSpec:
		// - type declarations
		ast.Walk(&u, n.Type)
	}
	return p
}

type usedWalker Package

// Walks through the AST marking used identifiers.
func (p *usedWalker) Visit(node ast.Node) ast.Visitor {
	// just be stupid and mark all *ast.Ident
	switch n := node.(type) {
	case *ast.Ident:
		p.used[n.Name] = true
	}
	return p
}
