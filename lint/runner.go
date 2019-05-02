package lint

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/types/objectpath"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/internal/cache"
	"honnef.co/go/tools/loader"
)

// If enabled, abuse of the go/analysis API will lead to panics
const sanityCheck = true

// OPT(dh): for a dependency tree A->B->C->D, if we have cached data
// for B, there should be no need to load C and D individually. Go's
// export data for B contains all the data we need on types, and our
// fact cache could store the union of B, C and D in B.
//
// This may change unused's behavior, however, as it may observe fewer
// interfaces from transitive dependencies.

type Package struct {
	*packages.Package
	Imports    map[string]*Package
	initial    bool
	fromSource bool
	hash       string

	resultsMu sync.Mutex
	results   []*result

	cfg      *config.Config
	gen      map[string]bool
	problems []Problem
	ignores  []Ignore
	errs     []error

	// these slices are indexed by analysis
	facts    []map[types.Object][]analysis.Fact
	pkgFacts [][]analysis.Fact
}

type result struct {
	v     interface{}
	err   error
	ready chan struct{}
}

type buildResult struct {
	done chan struct{}
}

type Runner struct {
	ld    loader.Loader
	cache *cache.Cache

	builtMu sync.Mutex
	built   map[*Package]*buildResult

	analyzerIDs analyzerIDs
}

type analyzerIDs struct {
	m map[*analysis.Analyzer]int
}

func (ids analyzerIDs) get(a *analysis.Analyzer) int {
	return ids.m[a]
}

type Fact struct {
	PkgPath string
	ObjPath string
	Fact    analysis.Fact
}

type newFact struct {
	obj  types.Object
	fact analysis.Fact
}

type analysisAction struct {
	analyzer   *analysis.Analyzer
	analyzerID int
	pkg        *Package
	newFacts   []newFact
	problems   []Problem

	pkgFacts map[*types.Package][]analysis.Fact
}

func (ac *analysisAction) allObjectFacts() []analysis.ObjectFact {
	out := make([]analysis.ObjectFact, 0, len(ac.pkg.facts[ac.analyzerID]))
	for obj, facts := range ac.pkg.facts[ac.analyzerID] {
		for _, fact := range facts {
			out = append(out, analysis.ObjectFact{
				Object: obj,
				Fact:   fact,
			})
		}
	}
	return out
}

func (ac *analysisAction) allPackageFacts() []analysis.PackageFact {
	out := make([]analysis.PackageFact, 0, len(ac.pkgFacts))
	for pkg, facts := range ac.pkgFacts {
		for _, fact := range facts {
			out = append(out, analysis.PackageFact{
				Package: pkg,
				Fact:    fact,
			})
		}
	}
	return out
}

func (ac *analysisAction) importObjectFact(obj types.Object, fact analysis.Fact) bool {
	if sanityCheck && len(ac.analyzer.FactTypes) == 0 {
		panic("analysis doesn't export any facts")
	}
	for _, f := range ac.pkg.facts[ac.analyzerID][obj] {
		if reflect.TypeOf(f) == reflect.TypeOf(fact) {
			reflect.ValueOf(fact).Elem().Set(reflect.ValueOf(f).Elem())
			return true
		}
	}
	return false
}

func (ac *analysisAction) importPackageFact(pkg *types.Package, fact analysis.Fact) bool {
	if sanityCheck && len(ac.analyzer.FactTypes) == 0 {
		panic("analysis doesn't export any facts")
	}
	for _, f := range ac.pkgFacts[pkg] {
		if reflect.TypeOf(f) == reflect.TypeOf(fact) {
			reflect.ValueOf(fact).Elem().Set(reflect.ValueOf(f).Elem())
			return true
		}
	}
	return false
}

func (ac *analysisAction) exportObjectFact(obj types.Object, fact analysis.Fact) {
	if sanityCheck && len(ac.analyzer.FactTypes) == 0 {
		panic("analysis doesn't export any facts")
	}
	ac.pkg.facts[ac.analyzerID][obj] = append(ac.pkg.facts[ac.analyzerID][obj], fact)
}

func (ac *analysisAction) exportPackageFact(fact analysis.Fact) {
	if sanityCheck && len(ac.analyzer.FactTypes) == 0 {
		panic("analysis doesn't export any facts")
	}
	ac.pkgFacts[ac.pkg.Types] = append(ac.pkgFacts[ac.pkg.Types], fact)
	ac.newFacts = append(ac.newFacts, newFact{nil, fact})
}

func (ac *analysisAction) report(pass *analysis.Pass, d analysis.Diagnostic) {
	p := Problem{
		Pos:     DisplayPosition(pass.Fset, d.Pos),
		Message: d.Message,
		Check:   pass.Analyzer.Name,
	}
	ac.problems = append(ac.problems, p)
}

func (r *Runner) runAnalysis(ac *analysisAction) (ret interface{}, err error) {
	ac.pkg.resultsMu.Lock()
	res := ac.pkg.results[r.analyzerIDs.get(ac.analyzer)]
	if res != nil {
		ac.pkg.resultsMu.Unlock()
		<-res.ready
		return res.v, res.err
	} else {
		res = &result{
			ready: make(chan struct{}),
		}
		ac.pkg.results[r.analyzerIDs.get(ac.analyzer)] = res
		ac.pkg.resultsMu.Unlock()

		defer func() {
			res.v = ret
			res.err = err
			close(res.ready)
		}()

		// Package may be a dependency or a package the user requested
		// Facts for a dependency may be cached or not
		// Diagnostics for a user package may be cached or not (not yet)
		// When we have to analyze a package, we have to analyze it with all dependencies.

		pass := new(analysis.Pass)
		*pass = analysis.Pass{
			Analyzer: ac.analyzer,
			Fset:     ac.pkg.Fset,
			Files:    ac.pkg.Syntax,
			// type information may be nil or may be populated. if it is
			// nil, it will get populated later.
			Pkg:               ac.pkg.Types,
			TypesInfo:         ac.pkg.TypesInfo,
			TypesSizes:        ac.pkg.TypesSizes,
			ResultOf:          map[*analysis.Analyzer]interface{}{},
			ImportObjectFact:  ac.importObjectFact,
			ImportPackageFact: ac.importPackageFact,
			ExportObjectFact:  ac.exportObjectFact,
			ExportPackageFact: ac.exportPackageFact,
			Report: func(d analysis.Diagnostic) {
				ac.report(pass, d)
			},
			AllObjectFacts:  ac.allObjectFacts,
			AllPackageFacts: ac.allPackageFacts,
		}

		if !ac.pkg.initial {
			// Don't report problems in dependencies
			pass.Report = func(analysis.Diagnostic) {}
		}
		return r.runAnalysisUser(pass, ac)
	}
}

func (r *Runner) loadCachedFacts(a *analysis.Analyzer, pkg *Package) ([]Fact, bool) {
	if len(a.FactTypes) == 0 {
		return nil, true
	}

	var facts []Fact
	// Look in the cache for facts
	aID, err := passActionID(pkg, a)
	if err != nil {
		return nil, false
	}
	aID = cache.Subkey(aID, "facts")
	b, _, err := r.cache.GetBytes(aID)
	if err != nil {
		// No cached facts, analyse this package like a user-provided one, but ignore diagnostics
		return nil, false
	}

	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&facts); err != nil {
		// Cached facts are broken, analyse this package like a user-provided one, but ignore diagnostics
		return nil, false
	}
	return facts, true
}

type dependencyError struct {
	dep string
	err error
}

func (err dependencyError) nested() dependencyError {
	if o, ok := err.err.(dependencyError); ok {
		return o.nested()
	}
	return err
}

func (err dependencyError) Error() string {
	if o, ok := err.err.(dependencyError); ok {
		return o.Error()
	}
	return fmt.Sprintf("error running dependency %s: %s", err.dep, err.err)
}

func (r *Runner) makeAnalysisAction(a *analysis.Analyzer, pkg *Package) *analysisAction {
	aid := r.analyzerIDs.get(a)
	ac := &analysisAction{
		analyzer:   a,
		analyzerID: aid,
		pkg:        pkg,
	}

	if len(a.FactTypes) == 0 {
		return ac
	}

	// Merge all package facts of dependencies
	ac.pkgFacts = map[*types.Package][]analysis.Fact{}
	seen := map[*Package]struct{}{}
	var dfs func(*Package)
	dfs = func(pkg *Package) {
		if _, ok := seen[pkg]; ok {
			return
		}
		seen[pkg] = struct{}{}
		s := pkg.pkgFacts[aid]
		ac.pkgFacts[pkg.Types] = s[0:len(s):len(s)]
		for _, imp := range pkg.Imports {
			dfs(imp)
		}
	}
	dfs(pkg)

	return ac
}

func (r *Runner) runAnalysisUser(pass *analysis.Pass, ac *analysisAction) (interface{}, error) {
	if !ac.pkg.fromSource {
		panic(fmt.Sprintf("internal error: %s was not loaded from source", ac.pkg))
	}

	// User-provided package, analyse it
	// First analyze it with dependencies
	var req []*analysis.Analyzer
	req = append(req, ac.analyzer.Requires...)
	if pass.Analyzer != IsGeneratedAnalyzer && pass.Analyzer != config.Analyzer {
		// Ensure all packages have the generated map and config. This is
		// required by interna of the runner. Analyses that themselves
		// make use of either have an explicit dependency so that other
		// runners work correctly, too.
		req = append(req, IsGeneratedAnalyzer, config.Analyzer)
	}
	for _, req := range req {
		acReq := r.makeAnalysisAction(req, ac.pkg)
		ret, err := r.runAnalysis(acReq)
		if err != nil {
			// We couldn't run a dependency, no point in going on
			return nil, dependencyError{req.Name, err}
		}

		pass.ResultOf[req] = ret
	}

	// Then with this analyzer
	ret, err := ac.analyzer.Run(pass)
	if err != nil {
		return nil, err
	}

	if len(ac.analyzer.FactTypes) > 0 {
		// Merge new facts into the package.
		for _, fact := range ac.newFacts {
			if fact.obj == nil {
				id := r.analyzerIDs.get(ac.analyzer)
				ac.pkg.pkgFacts[id] = append(ac.pkg.pkgFacts[id], fact.fact)
			} else {
				panic("unexpected new object fact")
			}
		}

		// Persist facts to cache
		facts := make([]Fact, 0, len(ac.newFacts))
		for _, fact := range ac.newFacts {
			if fact.obj == nil {
				facts = append(facts, Fact{fact.obj.Pkg().Path(), "", fact.fact})
			} else {
				path, err := objectpath.For(fact.obj)
				if err != nil {
					continue
				}
				facts = append(facts, Fact{fact.obj.Pkg().Path(), string(path), fact.fact})
			}
		}
		buf := &bytes.Buffer{}
		if err := gob.NewEncoder(buf).Encode(facts); err != nil {
			return nil, err
		}
		aID, err := passActionID(ac.pkg, ac.analyzer)
		if err != nil {
			return nil, err
		}
		aID = cache.Subkey(aID, "facts")
		if err := r.cache.PutBytes(aID, buf.Bytes()); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func NewRunner() (*Runner, error) {
	cache, err := cache.Default()
	if err != nil {
		return nil, err
	}

	return &Runner{
		cache: cache,
		built: map[*Package]*buildResult{},
	}, nil
}

func (r *Runner) Run(cfg *packages.Config, patterns []string, analyzers []*analysis.Analyzer) ([]*Package, error) {
	r.analyzerIDs = analyzerIDs{m: map[*analysis.Analyzer]int{}}
	id := 0
	seen := map[*analysis.Analyzer]struct{}{}
	var dfs func(a *analysis.Analyzer)
	dfs = func(a *analysis.Analyzer) {
		if _, ok := seen[a]; ok {
			return
		}
		seen[a] = struct{}{}
		r.analyzerIDs.m[a] = id
		id++
		for _, f := range a.FactTypes {
			gob.Register(f)
		}
		for _, req := range a.Requires {
			dfs(req)
		}
	}
	for _, a := range analyzers {
		dfs(a)
	}

	var dcfg packages.Config
	if cfg != nil {
		dcfg = *cfg
	}
	loaded, err := r.ld.Graph(dcfg, patterns...)
	if err != nil {
		return nil, err
	}

	defer r.cache.Trim()

	m := map[*packages.Package]*Package{}
	packages.Visit(loaded, nil, func(l *packages.Package) {
		m[l] = &Package{
			Package:  l,
			Imports:  map[string]*Package{},
			results:  make([]*result, len(r.analyzerIDs.m)),
			facts:    make([]map[types.Object][]analysis.Fact, len(r.analyzerIDs.m)),
			pkgFacts: make([][]analysis.Fact, len(r.analyzerIDs.m)),
		}
		for i := range m[l].facts {
			m[l].facts[i] = map[types.Object][]analysis.Fact{}
		}
		for _, err := range l.Errors {
			m[l].errs = append(m[l].errs, err)
		}
		for k, v := range l.Imports {
			m[l].Imports[k] = m[v]
		}

		m[l].hash, err = packageHash(m[l])
		if err != nil {
			m[l].errs = append(m[l].errs, err)
		}
	})
	pkgs := make([]*Package, len(loaded))
	for i, l := range loaded {
		pkgs[i] = m[l]
		pkgs[i].initial = true
	}

	var wg sync.WaitGroup
	wg.Add(len(pkgs))
	// OPT(dh): The ideal number of parallel jobs depends on the shape
	// of the graph. We may risk having one goroutine doing work and
	// all other goroutines being blocked on its completion. At the
	// same time, Go dependency graphs aren't always very amiable
	// towards parallelism. For example, on the standard library, we
	// only achieve about 400% CPU usage (out of a possible 800% on
	// this machine), and only 2x scaling.
	sem := make(chan struct{}, runtime.GOMAXPROCS(-1))
	for _, pkg := range pkgs {
		pkg := pkg
		sem <- struct{}{}
		go func() {
			r.processPkg(pkg, analyzers)
			<-sem
			wg.Done()
		}()
	}
	wg.Wait()

	return pkgs, nil
}

var posRe = regexp.MustCompile(`^(.+?):(\d+)(?::(\d+)?)?$`)

func parsePos(pos string) token.Position {
	if pos == "-" || pos == "" {
		return token.Position{}
	}
	parts := posRe.FindStringSubmatch(pos)
	if parts == nil {
		panic(fmt.Sprintf("internal error: malformed position %q", pos))
	}
	file := parts[1]
	line, _ := strconv.Atoi(parts[2])
	col, _ := strconv.Atoi(parts[3])
	return token.Position{
		Filename: file,
		Line:     line,
		Column:   col,
	}
}

// loadPkg loads a Go package. If the package is in the set of initial
// packages, it will be loaded from source, otherwise it will be
// loaded from export data. In the case that the package was loaded
// from export data, cached facts will also be loaded.
//
// Currently, only cached facts for this package will be loaded, not
// for any of its dependencies.
func (r *Runner) loadPkg(pkg *Package, analyzers []*analysis.Analyzer) error {
	if pkg.Types != nil {
		panic(fmt.Sprintf("internal error: %s has already been loaded", pkg.Package))
	}

	// Load type information
	if pkg.initial {
		// Load package from source
		pkg.fromSource = true
		return r.ld.LoadFromSource(pkg.Package)
	}

	// Load package from export data
	if err := r.ld.LoadFromExport(pkg.Package); err != nil {
		// We asked Go to give us up to date export data, yet
		// we can't load it. There must be something wrong.
		//
		// Attempt loading from source. This should fail (because
		// otherwise there would be export data); we just want to
		// get the compile errors. If loading from source succeeds
		// we discard the result, anyway. Otherwise we'll fail
		// when trying to reload from export data later.
		//
		// FIXME(dh): we no longer reload from export data, so
		// theoretically we should be able to continue
		pkg.fromSource = true
		if err := r.ld.LoadFromSource(pkg.Package); err != nil {
			return err
		}
		// Make sure this package can't be imported successfully
		pkg.Package.Errors = append(pkg.Package.Errors, packages.Error{
			Pos:  "-",
			Msg:  fmt.Sprintf("could not load export data: %s", err),
			Kind: packages.ParseError,
		})
		return fmt.Errorf("could not load export data: %s", err)
	}

	failed := false
	seen := make([]bool, len(r.analyzerIDs.m))
	var dfs func(*analysis.Analyzer)
	dfs = func(a *analysis.Analyzer) {
		if seen[r.analyzerIDs.get(a)] {
			return
		}
		seen[r.analyzerIDs.get(a)] = true

		if len(a.FactTypes) > 0 {
			facts, ok := r.loadCachedFacts(a, pkg)
			if !ok {
				failed = true
				return
			}

			for _, f := range facts {
				if pkg.PkgPath != f.PkgPath {
					// TODO(dh): for now we load all packages in the
					// dependency graph and don't utilize the fact
					// that a package contains all the facts of its
					// dependencies.
					continue
				}
				if f.ObjPath == "" {
					// This is a package fact
					pkg.pkgFacts[r.analyzerIDs.get(a)] = append(pkg.pkgFacts[r.analyzerIDs.get(a)], f.Fact)
					continue
				}
				obj, err := objectpath.Object(pkg.Types, objectpath.Path(f.ObjPath))
				if err != nil {
					// Be lenient about these errors. For example, when
					// analysing io/ioutil from source, we may get a fact
					// for methods on the devNull type, and objectpath
					// will happily create a path for them. However, when
					// we later load io/ioutil from export data, the path
					// no longer resolves.
					//
					// If an exported type embeds the unexported type,
					// then (part of) the unexported type will become part
					// of the type information and our path will resolve
					// again.
					continue
				}
				pkg.facts[r.analyzerIDs.get(a)][obj] = append(pkg.facts[r.analyzerIDs.get(a)][obj], f.Fact)
			}
		}

		for _, req := range a.Requires {
			dfs(req)
		}
	}
	for _, a := range analyzers {
		dfs(a)
	}

	if failed {
		pkg.fromSource = true
		// XXX we added facts to the maps, we need to get rid of those
		return r.ld.LoadFromSource(pkg.Package)
	}

	return nil
}

type analysisError struct {
	analyzer *analysis.Analyzer
	pkg      *Package
	err      error
}

func (err analysisError) Error() string {
	return fmt.Sprintf("error running analyzer %s on %s: %s", err.analyzer, err.pkg, err.err)
}

// processPkg processes a package. This involves loading the package,
// either from export data or from source. For packages loaded from
// source, the provides analyzers will be run on the package.
func (r *Runner) processPkg(pkg *Package, analyzers []*analysis.Analyzer) {
	r.builtMu.Lock()
	res := r.built[pkg]
	if res != nil {
		r.builtMu.Unlock()
		<-res.done
		return
	}

	res = &buildResult{done: make(chan struct{})}
	r.built[pkg] = res
	r.builtMu.Unlock()

	defer func() {
		// Clear information we no longer need. Make sure to do this
		// when returning from processPkg so that we clear
		// dependencies, not just initial packages.
		pkg.TypesInfo = nil
		pkg.Syntax = nil
		pkg.results = nil
		close(res.done)
	}()

	if len(pkg.errs) != 0 {
		return
	}

	for _, imp := range pkg.Imports {
		r.processPkg(imp, analyzers)
		if len(imp.errs) > 0 {
			if imp.initial {
				pkg.errs = append(pkg.errs, fmt.Errorf("could not analyze dependency %s of %s", imp, pkg))
			} else {
				var s string
				for _, err := range imp.errs {
					s += "\n\t" + err.Error()
				}
				pkg.errs = append(pkg.errs, fmt.Errorf("could not analyze dependency %s of %s: %s", imp, pkg, s))
			}
			return
		}
	}
	if pkg.PkgPath == "unsafe" {
		pkg.Types = types.Unsafe
		return
	}

	if err := r.loadPkg(pkg, analyzers); err != nil {
		pkg.errs = append(pkg.errs, err)
		return
	}

	// A package's object facts is the union of all of its dependencies.
	for _, imp := range pkg.Imports {
		for ai, m := range imp.facts {
			for obj, facts := range m {
				pkg.facts[ai][obj] = facts[0:len(facts):len(facts)]
			}
		}
	}

	if !pkg.fromSource {
		// Nothing left to do for the package.
		return
	}

	// Run analyses on initial packages and those missing facts
	var wg sync.WaitGroup
	wg.Add(len(analyzers))
	errs := make([]error, len(analyzers))
	var acs []*analysisAction
	for i, a := range analyzers {
		i := i
		a := a
		ac := r.makeAnalysisAction(a, pkg)
		acs = append(acs, ac)
		go func() {
			defer wg.Done()
			// Only initial packages and packages with missing
			// facts will have been loaded from source.
			if pkg.initial || r.hasFacts(a) {
				if _, err := r.runAnalysis(ac); err != nil {
					errs[i] = analysisError{a, pkg, err}
					return
				}
			}
		}()
	}
	wg.Wait()

	depErrors := map[dependencyError]int{}
	for _, err := range errs {
		if err == nil {
			continue
		}
		switch err := err.(type) {
		case analysisError:
			switch err := err.err.(type) {
			case dependencyError:
				depErrors[err.nested()]++
			default:
				pkg.errs = append(pkg.errs, err)
			}
		default:
			pkg.errs = append(pkg.errs, err)
		}
	}
	for err, count := range depErrors {
		pkg.errs = append(pkg.errs,
			fmt.Errorf("could not run %s@%s, preventing %d analyzers from running: %s", err.dep, pkg, count, err.err))
	}

	// We can't process ignores at this point because `unused` needs
	// to see more than one package to make its decision.
	ignores, problems := parseDirectives(pkg.Package)
	pkg.ignores = append(pkg.ignores, ignores...)
	pkg.problems = append(pkg.problems, problems...)
	for _, ac := range acs {
		pkg.problems = append(pkg.problems, ac.problems...)
	}
	if pkg.results[r.analyzerIDs.get(config.Analyzer)].v != nil {
		pkg.cfg = pkg.results[r.analyzerIDs.get(config.Analyzer)].v.(*config.Config)
	}
	pkg.gen = pkg.results[r.analyzerIDs.get(IsGeneratedAnalyzer)].v.(map[string]bool)

	// In a previous version of the code, we would throw away all type
	// information and reload it from export data. That was
	// nonsensical. The *types.Package doesn't keep any information
	// live that export data wouldn't also. We only need to discard
	// the AST and the TypesInfo maps; that happens after we return
	// from processPkg.
}

func (r *Runner) hasFacts(a *analysis.Analyzer) bool {
	ret := false
	seen := make([]bool, len(r.analyzerIDs.m))
	var dfs func(*analysis.Analyzer)
	dfs = func(a *analysis.Analyzer) {
		if seen[r.analyzerIDs.get(a)] {
			return
		}
		seen[r.analyzerIDs.get(a)] = true
		if len(a.FactTypes) > 0 {
			ret = true
		}
		for _, req := range a.Requires {
			if ret {
				break
			}
			dfs(req)
		}
	}
	dfs(a)
	return ret
}

func parseDirective(s string) (cmd string, args []string) {
	if !strings.HasPrefix(s, "//lint:") {
		return "", nil
	}
	s = strings.TrimPrefix(s, "//lint:")
	fields := strings.Split(s, " ")
	return fields[0], fields[1:]
}

func parseDirectives(pkg *packages.Package) ([]Ignore, []Problem) {
	var ignores []Ignore
	var problems []Problem

	for _, f := range pkg.Syntax {
		found := false
	commentLoop:
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if strings.Contains(c.Text, "//lint:") {
					found = true
					break commentLoop
				}
			}
		}
		if !found {
			continue
		}
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
								Pos:      DisplayPosition(pkg.Fset, c.Pos()),
								Message:  "malformed linter directive; missing the required reason field?",
								Severity: Error,
								Check:    "",
							}
							problems = append(problems, p)
							continue
						}
					default:
						// unknown directive, ignore
						continue
					}
					checks := strings.Split(args[0], ",")
					pos := DisplayPosition(pkg.Fset, node.Pos())
					var ig Ignore
					switch cmd {
					case "ignore":
						ig = &LineIgnore{
							File:   pos.Filename,
							Line:   pos.Line,
							Checks: checks,
							Pos:    c.Pos(),
						}
					case "file-ignore":
						ig = &FileIgnore{
							File:   pos.Filename,
							Checks: checks,
						}
					}
					ignores = append(ignores, ig)
				}
			}
		}
	}

	return ignores, problems
}

func packageHash(pkg *Package) (string, error) {
	key := cache.NewHash("package hash")
	fmt.Fprintf(key, "pkgpath %s\n", pkg.PkgPath)
	for _, f := range pkg.CompiledGoFiles {
		h, err := cache.FileHash(f)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(key, "file %s %x\n", f, h)
	}
	imps := make([]*Package, 0, len(pkg.Imports))
	for _, v := range pkg.Imports {
		imps = append(imps, v)
	}
	sort.Slice(imps, func(i, j int) bool {
		return imps[i].PkgPath < imps[j].PkgPath
	})
	for _, dep := range imps {
		if dep.PkgPath == "unsafe" {
			continue
		}

		fmt.Fprintf(key, "import %s %s\n", dep.PkgPath, dep.hash)
	}
	h := key.Sum()
	return hex.EncodeToString(h[:]), nil
}

func passActionID(pkg *Package, analyzer *analysis.Analyzer) (cache.ActionID, error) {
	key := cache.NewHash("action ID")
	fmt.Fprintf(key, "pkgpath %s\n", pkg.PkgPath)
	fmt.Fprintf(key, "pkghash %s\n", pkg.hash)
	fmt.Fprintf(key, "analyzer %s\n", analyzer.Name)

	return key.Sum(), nil
}
