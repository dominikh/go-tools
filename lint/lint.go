// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package lint provides the foundation for tools like gosimple.
package lint // import "honnef.co/go/tools/lint"

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/tools/go/packages"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/ssa"
	gossautil "honnef.co/go/tools/ssa/ssautil"
)

type Job struct {
	Program *Program

	checker  string
	check    string
	problems []Problem

	duration time.Duration
}

type Ignore interface {
	Match(p Problem) bool
}

type LineIgnore struct {
	File    string
	Line    int
	Checks  []string
	matched bool
	pos     token.Pos
}

func (li *LineIgnore) Match(p Problem) bool {
	if p.Position.Filename != li.File || p.Position.Line != li.Line {
		return false
	}
	for _, c := range li.Checks {
		if m, _ := filepath.Match(c, p.Check); m {
			li.matched = true
			return true
		}
	}
	return false
}

func (li *LineIgnore) String() string {
	matched := "not matched"
	if li.matched {
		matched = "matched"
	}
	return fmt.Sprintf("%s:%d %s (%s)", li.File, li.Line, strings.Join(li.Checks, ", "), matched)
}

type FileIgnore struct {
	File   string
	Checks []string
}

func (fi *FileIgnore) Match(p Problem) bool {
	if p.Position.Filename != fi.File {
		return false
	}
	for _, c := range fi.Checks {
		if m, _ := filepath.Match(c, p.Check); m {
			return true
		}
	}
	return false
}

type GlobIgnore struct {
	Pattern string
	Checks  []string
}

func (gi *GlobIgnore) Match(p Problem) bool {
	if gi.Pattern != "*" {
		pkgpath := p.Package.Types.Path()
		if strings.HasSuffix(pkgpath, "_test") {
			pkgpath = pkgpath[:len(pkgpath)-len("_test")]
		}
		name := filepath.Join(pkgpath, filepath.Base(p.Position.Filename))
		if m, _ := filepath.Match(gi.Pattern, name); !m {
			return false
		}
	}
	for _, c := range gi.Checks {
		if m, _ := filepath.Match(c, p.Check); m {
			return true
		}
	}
	return false
}

type Program struct {
	SSA              *ssa.Program
	InitialPackages  []*Pkg
	InitialFunctions []*ssa.Function
	AllPackages      []*packages.Package
	AllFunctions     []*ssa.Function
	Files            []*ast.File
	Info             *types.Info
	GoVersion        int

	tokenFileMap map[*token.File]*ast.File
	astFileMap   map[*ast.File]*Pkg

	packagesMap map[string]*packages.Package
}

func (p *Program) Fset() *token.FileSet {
	return p.InitialPackages[0].Fset
}

type Func func(*Job)

// Problem represents a problem in some source code.
type Problem struct {
	Position token.Position // position in source file
	Text     string         // the prose that describes the problem
	Check    string
	Checker  string
	Package  *Pkg
	Ignored  bool
}

func (p *Problem) String() string {
	if p.Check == "" {
		return p.Text
	}
	return fmt.Sprintf("%s (%s)", p.Text, p.Check)
}

type Checker interface {
	Name() string
	Prefix() string
	Init(*Program)
	Funcs() map[string]Func
}

// A Linter lints Go source code.
type Linter struct {
	Checkers      []Checker
	Ignores       []Ignore
	GoVersion     int
	ReturnIgnored bool

	MaxConcurrentJobs int
	PrintStats        bool

	automaticIgnores []Ignore
}

func (l *Linter) ignore(p Problem) bool {
	ignored := false
	for _, ig := range l.automaticIgnores {
		// We cannot short-circuit these, as we want to record, for
		// each ignore, whether it matched or not.
		if ig.Match(p) {
			ignored = true
		}
	}
	if ignored {
		// no need to execute other ignores if we've already had a
		// match.
		return true
	}
	for _, ig := range l.Ignores {
		// We can short-circuit here, as we aren't tracking any
		// information.
		if ig.Match(p) {
			return true
		}
	}

	return false
}

func (prog *Program) File(node Positioner) *ast.File {
	return prog.tokenFileMap[prog.SSA.Fset.File(node.Pos())]
}

func (j *Job) File(node Positioner) *ast.File {
	return j.Program.File(node)
}

func parseDirective(s string) (cmd string, args []string) {
	if !strings.HasPrefix(s, "//lint:") {
		return "", nil
	}
	s = strings.TrimPrefix(s, "//lint:")
	fields := strings.Split(s, " ")
	return fields[0], fields[1:]
}

type PerfStats struct {
	PackageLoading time.Duration
	SSABuild       time.Duration
	OtherInitWork  time.Duration
	CheckerInits   map[string]time.Duration
	Jobs           []JobStat
}

type JobStat struct {
	Job      string
	Duration time.Duration
}

func (stats *PerfStats) Print(w io.Writer) {
	fmt.Fprintln(w, "Package loading:", stats.PackageLoading)
	fmt.Fprintln(w, "SSA build:", stats.SSABuild)
	fmt.Fprintln(w, "Other init work:", stats.OtherInitWork)

	fmt.Fprintln(w, "Checker inits:")
	for checker, d := range stats.CheckerInits {
		fmt.Fprintf(w, "\t%s: %s\n", checker, d)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Jobs:")
	sort.Slice(stats.Jobs, func(i, j int) bool {
		return stats.Jobs[i].Duration < stats.Jobs[j].Duration
	})
	var total time.Duration
	for _, job := range stats.Jobs {
		fmt.Fprintf(w, "\t%s: %s\n", job.Job, job.Duration)
		total += job.Duration
	}
	fmt.Fprintf(w, "\tTotal: %s\n", total)
}

func (l *Linter) Lint(initial []*packages.Package, stats *PerfStats) []Problem {
	allPkgs := allPackages(initial)
	t := time.Now()
	ssaprog, _ := gossautil.Packages(allPkgs, ssa.GlobalDebug)
	ssaprog.Build()
	if stats != nil {
		stats.SSABuild = time.Since(t)
	}

	t = time.Now()
	pkgMap := map[*ssa.Package]*Pkg{}
	var pkgs []*Pkg
	for _, pkg := range initial {
		ssapkg := ssaprog.Package(pkg.Types)
		var cfg config.Config
		if len(pkg.GoFiles) != 0 {
			path := pkg.GoFiles[0]
			dir := filepath.Dir(path)
			var err error
			// OPT(dh): we're rebuilding the entire config tree for
			// each package. for example, if we check a/b/c and
			// a/b/c/d, we'll process a, a/b, a/b/c, a, a/b, a/b/c,
			// a/b/c/d – we should cache configs per package and only
			// load the new levels.
			cfg, err = config.Load(dir)
			if err != nil {
				// FIXME(dh): we couldn't load the config, what are we
				// supposed to do? probably tell the user somehow
			}
		}

		pkg := &Pkg{
			SSA:     ssapkg,
			Package: pkg,
			Config:  cfg,
		}
		pkgMap[ssapkg] = pkg
		pkgs = append(pkgs, pkg)
	}

	prog := &Program{
		SSA:             ssaprog,
		InitialPackages: pkgs,
		AllPackages:     allPkgs,
		Info:            &types.Info{},
		GoVersion:       l.GoVersion,
		tokenFileMap:    map[*token.File]*ast.File{},
		astFileMap:      map[*ast.File]*Pkg{},
	}
	prog.packagesMap = map[string]*packages.Package{}
	for _, pkg := range allPkgs {
		prog.packagesMap[pkg.Types.Path()] = pkg
	}

	isInitial := map[*types.Package]struct{}{}
	for _, pkg := range pkgs {
		isInitial[pkg.Types] = struct{}{}
	}
	for fn := range gossautil.AllFunctions(ssaprog) {
		if fn.Pkg == nil {
			continue
		}
		prog.AllFunctions = append(prog.AllFunctions, fn)
		if _, ok := isInitial[fn.Pkg.Pkg]; ok {
			prog.InitialFunctions = append(prog.InitialFunctions, fn)
		}
	}
	for _, pkg := range pkgs {
		prog.Files = append(prog.Files, pkg.Syntax...)

		ssapkg := ssaprog.Package(pkg.Types)
		for _, f := range pkg.Syntax {
			prog.astFileMap[f] = pkgMap[ssapkg]
		}
	}

	for _, pkg := range allPkgs {
		for _, f := range pkg.Syntax {
			tf := pkg.Fset.File(f.Pos())
			prog.tokenFileMap[tf] = f
		}
	}

	var out []Problem
	l.automaticIgnores = nil
	for _, pkg := range initial {
		for _, f := range pkg.Syntax {
			cm := ast.NewCommentMap(pkg.Fset, f, f.Comments)
			for node, cgs := range cm {
				for _, cg := range cgs {
					for _, c := range cg.List {
						if !strings.HasPrefix(c.Text, "//lint:") {
							continue
						}
						cmd, args := parseDirective(c.Text)
						switch cmd {
						case "ignore", "file-ignore":
							if len(args) < 2 {
								// FIXME(dh): this causes duplicated warnings when using megacheck
								p := Problem{
									Position: prog.DisplayPosition(c.Pos()),
									Text:     "malformed linter directive; missing the required reason field?",
									Check:    "",
									Checker:  "lint",
									Package:  nil,
								}
								out = append(out, p)
								continue
							}
						default:
							// unknown directive, ignore
							continue
						}
						checks := strings.Split(args[0], ",")
						pos := prog.DisplayPosition(node.Pos())
						var ig Ignore
						switch cmd {
						case "ignore":
							ig = &LineIgnore{
								File:   pos.Filename,
								Line:   pos.Line,
								Checks: checks,
								pos:    c.Pos(),
							}
						case "file-ignore":
							ig = &FileIgnore{
								File:   pos.Filename,
								Checks: checks,
							}
						}
						l.automaticIgnores = append(l.automaticIgnores, ig)
					}
				}
			}
		}
	}

	sizes := struct {
		types      int
		defs       int
		uses       int
		implicits  int
		selections int
		scopes     int
	}{}
	for _, pkg := range pkgs {
		sizes.types += len(pkg.TypesInfo.Types)
		sizes.defs += len(pkg.TypesInfo.Defs)
		sizes.uses += len(pkg.TypesInfo.Uses)
		sizes.implicits += len(pkg.TypesInfo.Implicits)
		sizes.selections += len(pkg.TypesInfo.Selections)
		sizes.scopes += len(pkg.TypesInfo.Scopes)
	}
	prog.Info.Types = make(map[ast.Expr]types.TypeAndValue, sizes.types)
	prog.Info.Defs = make(map[*ast.Ident]types.Object, sizes.defs)
	prog.Info.Uses = make(map[*ast.Ident]types.Object, sizes.uses)
	prog.Info.Implicits = make(map[ast.Node]types.Object, sizes.implicits)
	prog.Info.Selections = make(map[*ast.SelectorExpr]*types.Selection, sizes.selections)
	prog.Info.Scopes = make(map[ast.Node]*types.Scope, sizes.scopes)
	for _, pkg := range pkgs {
		for k, v := range pkg.TypesInfo.Types {
			prog.Info.Types[k] = v
		}
		for k, v := range pkg.TypesInfo.Defs {
			prog.Info.Defs[k] = v
		}
		for k, v := range pkg.TypesInfo.Uses {
			prog.Info.Uses[k] = v
		}
		for k, v := range pkg.TypesInfo.Implicits {
			prog.Info.Implicits[k] = v
		}
		for k, v := range pkg.TypesInfo.Selections {
			prog.Info.Selections[k] = v
		}
		for k, v := range pkg.TypesInfo.Scopes {
			prog.Info.Scopes[k] = v
		}
	}

	if stats != nil {
		stats.OtherInitWork = time.Since(t)
	}

	for _, checker := range l.Checkers {
		t := time.Now()
		checker.Init(prog)
		if stats != nil {
			stats.CheckerInits[checker.Name()] = time.Since(t)
		}
	}

	var jobs []*Job
	var allKeys []string
	funcs := map[string]Func{}
	for _, checker := range l.Checkers {
		m := checker.Funcs()
		var keys []string
		for k, v := range m {
			funcs[k] = v
			keys = append(keys, k)
		}
		sort.Strings(keys)
		allKeys = append(allKeys, keys...)

		for _, k := range keys {
			j := &Job{
				Program: prog,
				checker: checker.Name(),
				check:   k,
			}
			jobs = append(jobs, j)
		}
	}

	max := len(jobs)
	if l.MaxConcurrentJobs > 0 {
		max = l.MaxConcurrentJobs
	}

	sem := make(chan struct{}, max)
	wg := &sync.WaitGroup{}
	for _, j := range jobs {
		wg.Add(1)
		go func(j *Job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			fn := funcs[j.check]
			if fn == nil {
				return
			}
			t := time.Now()
			fn(j)
			j.duration = time.Since(t)
		}(j)
	}
	wg.Wait()

	for _, j := range jobs {
		if stats != nil {
			stats.Jobs = append(stats.Jobs, JobStat{j.check, j.duration})
		}
		for _, p := range j.problems {
			// OPT(dh): this entire computation could be cached per package
			allowedChecks := map[string]bool{}
			var enabled, disabled []string
			// TODO(dh): we don't want to hard-code a list of supported checkers.
			switch p.Checker {
			case "staticcheck":
				enabled = p.Package.Config.Staticcheck.EnabledChecks
				disabled = p.Package.Config.Staticcheck.DisabledChecks
			case "simple":
				enabled = p.Package.Config.Simple.EnabledChecks
				disabled = p.Package.Config.Simple.DisabledChecks
			case "unused":
				enabled = p.Package.Config.Unused.EnabledChecks
				disabled = p.Package.Config.Unused.DisabledChecks
			case "errcheck":
				enabled = p.Package.Config.Errcheck.EnabledChecks
				disabled = p.Package.Config.Errcheck.DisabledChecks
			case "stylecheck":
				enabled = p.Package.Config.Stylecheck.EnabledChecks
				disabled = p.Package.Config.Stylecheck.DisabledChecks
			default:
				enabled = []string{"all"}
				disabled = nil
			}
			for _, c := range enabled {
				if c == "all" {
					for _, c := range allKeys {
						allowedChecks[c] = true
					}
				} else {
					allowedChecks[c] = true
				}
			}
			for _, c := range disabled {
				if c == "all" {
					allowedChecks = nil
				} else {
					// TODO(dh): support globs in check white/blacklist
					delete(allowedChecks, c)
				}
			}
			p.Ignored = l.ignore(p)
			// TODO(dh): support globs in check white/blacklist
			// OPT(dh): this approach doesn't actually disable checks,
			// it just discards their results. For the moment, that's
			// fine. None of our checks are super expensive. In the
			// future, we may want to provide opt-in expensive
			// analysis, which shouldn't run at all. It may be easiest
			// to implement this in the individual checks.
			if (l.ReturnIgnored || !p.Ignored) && allowedChecks[p.Check] {
				out = append(out, p)
			}
		}
	}

	for _, ig := range l.automaticIgnores {
		ig, ok := ig.(*LineIgnore)
		if !ok {
			continue
		}
		if ig.matched {
			continue
		}
		for _, c := range ig.Checks {
			idx := strings.IndexFunc(c, func(r rune) bool {
				return unicode.IsNumber(r)
			})
			if idx == -1 {
				// malformed check name, backing out
				continue
			}

			responsible := false
			for _, checker := range l.Checkers {
				if c[:idx] == checker.Prefix() {
					// for this checker
					responsible = true
					break
				}
			}
			if !responsible {
				continue
			}
			p := Problem{
				Position: prog.DisplayPosition(ig.pos),
				Text:     "this linter directive didn't match anything; should it be removed?",
				Check:    "",
				Checker:  "lint",
				Package:  nil,
			}
			out = append(out, p)
		}
	}

	sort.Slice(out, func(i int, j int) bool {
		pi, pj := out[i].Position, out[j].Position

		if pi.Filename != pj.Filename {
			return pi.Filename < pj.Filename
		}
		if pi.Line != pj.Line {
			return pi.Line < pj.Line
		}
		if pi.Column != pj.Column {
			return pi.Column < pj.Column
		}

		return out[i].Text < out[j].Text
	})

	if len(out) < 2 {
		return out
	}

	uniq := make([]Problem, 0, len(out))
	uniq = append(uniq, out[0])
	prev := out[0]
	for _, p := range out[1:] {
		if prev.Position == p.Position && prev.Text == p.Text {
			continue
		}
		prev = p
		uniq = append(uniq, p)
	}

	if l.PrintStats && stats != nil {
		stats.Print(os.Stderr)
	}
	return uniq
}

func (prog *Program) Package(path string) *packages.Package {
	return prog.packagesMap[path]
}

// Pkg represents a package being linted.
type Pkg struct {
	SSA *ssa.Package
	*packages.Package
	Config config.Config
}

type Positioner interface {
	Pos() token.Pos
}

func (prog *Program) DisplayPosition(p token.Pos) token.Position {
	// Only use the adjusted position if it points to another Go file.
	// This means we'll point to the original file for cgo files, but
	// we won't point to a YACC grammar file.

	pos := prog.Fset().PositionFor(p, false)
	adjPos := prog.Fset().PositionFor(p, true)

	if filepath.Ext(adjPos.Filename) == ".go" {
		return adjPos
	}
	return pos
}

func (j *Job) Errorf(n Positioner, format string, args ...interface{}) *Problem {
	tf := j.Program.SSA.Fset.File(n.Pos())
	f := j.Program.tokenFileMap[tf]
	pkg := j.Program.astFileMap[f]

	pos := j.Program.DisplayPosition(n.Pos())
	problem := Problem{
		Position: pos,
		Text:     fmt.Sprintf(format, args...),
		Check:    j.check,
		Checker:  j.checker,
		Package:  pkg,
	}
	j.problems = append(j.problems, problem)
	return &j.problems[len(j.problems)-1]
}

func (j *Job) NodePackage(node Positioner) *Pkg {
	f := j.File(node)
	return j.Program.astFileMap[f]
}

func allPackages(pkgs []*packages.Package) []*packages.Package {
	all := map[*packages.Package]bool{}
	var wl []*packages.Package
	wl = append(wl, pkgs...)
	for len(wl) > 0 {
		pkg := wl[len(wl)-1]
		wl = wl[:len(wl)-1]
		if all[pkg] {
			continue
		}
		all[pkg] = true
		for _, imp := range pkg.Imports {
			wl = append(wl, imp)
		}
	}

	out := make([]*packages.Package, 0, len(all))
	for pkg := range all {
		out = append(out, pkg)
	}
	return out
}
