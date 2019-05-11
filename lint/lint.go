// Package lint provides the foundation for tools like staticcheck
package lint // import "honnef.co/go/tools/lint"

import (
	"bytes"
	"fmt"
	"go/scanner"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
	"honnef.co/go/tools/config"
)

type Ignore interface {
	Match(p Problem) bool
}

type LineIgnore struct {
	File    string
	Line    int
	Checks  []string
	Matched bool
	Pos     token.Pos
}

func (li *LineIgnore) Match(p Problem) bool {
	pos := p.Pos
	if pos.Filename != li.File || pos.Line != li.Line {
		return false
	}
	for _, c := range li.Checks {
		if m, _ := filepath.Match(c, p.Check); m {
			li.Matched = true
			return true
		}
	}
	return false
}

func (li *LineIgnore) String() string {
	matched := "not matched"
	if li.Matched {
		matched = "matched"
	}
	return fmt.Sprintf("%s:%d %s (%s)", li.File, li.Line, strings.Join(li.Checks, ", "), matched)
}

type FileIgnore struct {
	File   string
	Checks []string
}

func (fi *FileIgnore) Match(p Problem) bool {
	if p.Pos.Filename != fi.File {
		return false
	}
	for _, c := range fi.Checks {
		if m, _ := filepath.Match(c, p.Check); m {
			return true
		}
	}
	return false
}

type Severity uint8

const (
	Error Severity = iota
	Warning
	Ignored
)

// Problem represents a problem in some source code.
type Problem struct {
	Pos      token.Position
	Message  string
	Check    string
	Severity Severity
}

func (p *Problem) String() string {
	return fmt.Sprintf("%s (%s)", p.Message, p.Check)
}

// A Linter lints Go source code.
type Linter struct {
	Checkers           []*analysis.Analyzer
	CumulativeCheckers []CumulativeChecker
	GoVersion          int
	Config             config.Config
	Stats              Stats
}

type CumulativeChecker interface {
	Analyzer() *analysis.Analyzer
	Result() []types.Object
	ProblemObject(*token.FileSet, types.Object) Problem
}

func (l *Linter) Lint(cfg *packages.Config, patterns []string) ([]Problem, error) {
	var analyzers []*analysis.Analyzer
	analyzers = append(analyzers, l.Checkers...)
	for _, cum := range l.CumulativeCheckers {
		analyzers = append(analyzers, cum.Analyzer())
	}

	r, err := NewRunner(&l.Stats)
	if err != nil {
		return nil, err
	}

	pkgs, err := r.Run(cfg, patterns, analyzers)
	if err != nil {
		return nil, err
	}

	tpkgToPkg := map[*types.Package]*Package{}
	for _, pkg := range pkgs {
		tpkgToPkg[pkg.Types] = pkg

		for _, err := range pkg.errs {
			switch err := err.(type) {
			case types.Error:
				p := Problem{
					Pos:      err.Fset.PositionFor(err.Pos, false),
					Message:  err.Msg,
					Severity: Error,
					Check:    "compile",
				}
				pkg.problems = append(pkg.problems, p)
			case packages.Error:
				p := Problem{
					Pos:      parsePos(err.Pos),
					Message:  err.Msg,
					Severity: Error,
					Check:    "compile",
				}
				pkg.problems = append(pkg.problems, p)
			case scanner.ErrorList:
				for _, err := range err {
					p := Problem{
						Pos:      err.Pos,
						Message:  err.Msg,
						Severity: Error,
						Check:    "compile",
					}
					pkg.problems = append(pkg.problems, p)
				}
			case error:
				p := Problem{
					Pos:      token.Position{},
					Message:  err.Error(),
					Severity: Error,
					Check:    "compile",
				}
				pkg.problems = append(pkg.problems, p)
			}
		}
	}

	atomic.StoreUint64(&r.stats.State, StateCumulative)
	var problems []Problem
	for _, cum := range l.CumulativeCheckers {
		for _, res := range cum.Result() {
			pkg := tpkgToPkg[res.Pkg()]
			allowedChecks := FilterChecks(analyzers, pkg.cfg.Merge(l.Config).Checks)
			if allowedChecks[cum.Analyzer().Name] {
				pos := DisplayPosition(pkg.Fset, res.Pos())
				if pkg.gen[pos.Filename] {
					continue
				}
				p := cum.ProblemObject(pkg.Fset, res)
				problems = append(problems, p)
			}
		}
	}

	for _, pkg := range pkgs {
		for _, ig := range pkg.ignores {
			for i := range pkg.problems {
				p := &pkg.problems[i]
				if ig.Match(*p) {
					p.Severity = Ignored
				}
			}
			for i := range problems {
				p := &problems[i]
				if ig.Match(*p) {
					p.Severity = Ignored
				}
			}
		}

		if pkg.cfg == nil {
			// The package failed to load, otherwise we would have a
			// valid config. Pass through all errors.
			problems = append(problems, pkg.problems...)
		} else {
			for _, p := range pkg.problems {
				allowedChecks := FilterChecks(analyzers, pkg.cfg.Merge(l.Config).Checks)
				allowedChecks["compile"] = true
				if allowedChecks[p.Check] {
					problems = append(problems, p)
				}
			}
		}

		for _, ig := range pkg.ignores {
			ig, ok := ig.(*LineIgnore)
			if !ok {
				continue
			}
			if ig.Matched {
				continue
			}

			couldveMatched := false
			allowedChecks := FilterChecks(analyzers, pkg.cfg.Merge(l.Config).Checks)
			for _, c := range ig.Checks {
				if !allowedChecks[c] {
					continue
				}
				couldveMatched = true
				break
			}

			if !couldveMatched {
				// The ignored checks were disabled for the containing package.
				// Don't flag the ignore for not having matched.
				continue
			}
			p := Problem{
				Pos:     DisplayPosition(pkg.Fset, ig.Pos),
				Message: "this linter directive didn't match anything; should it be removed?",
				Check:   "",
			}
			problems = append(problems, p)
		}
	}

	if len(problems) == 0 {
		return nil, nil
	}

	sort.Slice(problems, func(i, j int) bool {
		pi := problems[i].Pos
		pj := problems[j].Pos

		if pi.Filename != pj.Filename {
			return pi.Filename < pj.Filename
		}
		if pi.Line != pj.Line {
			return pi.Line < pj.Line
		}
		if pi.Column != pj.Column {
			return pi.Column < pj.Column
		}

		return problems[i].Message < problems[j].Message
	})

	var out []Problem
	out = append(out, problems[0])
	for i, p := range problems[1:] {
		// We may encounter duplicate problems because one file
		// can be part of many packages.
		if problems[i] != p {
			out = append(out, p)
		}
	}
	return out, nil
}

func FilterChecks(allChecks []*analysis.Analyzer, checks []string) map[string]bool {
	// OPT(dh): this entire computation could be cached per package
	allowedChecks := map[string]bool{}

	for _, check := range checks {
		b := true
		if len(check) > 1 && check[0] == '-' {
			b = false
			check = check[1:]
		}
		if check == "*" || check == "all" {
			// Match all
			for _, c := range allChecks {
				allowedChecks[c.Name] = b
			}
		} else if strings.HasSuffix(check, "*") {
			// Glob
			prefix := check[:len(check)-1]
			isCat := strings.IndexFunc(prefix, func(r rune) bool { return unicode.IsNumber(r) }) == -1

			for _, c := range allChecks {
				idx := strings.IndexFunc(c.Name, func(r rune) bool { return unicode.IsNumber(r) })
				if isCat {
					// Glob is S*, which should match S1000 but not SA1000
					cat := c.Name[:idx]
					if prefix == cat {
						allowedChecks[c.Name] = b
					}
				} else {
					// Glob is S1*
					if strings.HasPrefix(c.Name, prefix) {
						allowedChecks[c.Name] = b
					}
				}
			}
		} else {
			// Literal check name
			allowedChecks[check] = b
		}
	}
	return allowedChecks
}

type Positioner interface {
	Pos() token.Pos
}

func DisplayPosition(fset *token.FileSet, p token.Pos) token.Position {
	// Only use the adjusted position if it points to another Go file.
	// This means we'll point to the original file for cgo files, but
	// we won't point to a YACC grammar file.

	pos := fset.PositionFor(p, false)
	adjPos := fset.PositionFor(p, true)

	if filepath.Ext(adjPos.Filename) == ".go" {
		return adjPos
	}
	return pos
}

var bufferPool = &sync.Pool{
	New: func() interface{} {
		buf := bytes.NewBuffer(nil)
		buf.Grow(64)
		return buf
	},
}

func FuncName(f *types.Func) string {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	if f.Type() != nil {
		sig := f.Type().(*types.Signature)
		if recv := sig.Recv(); recv != nil {
			buf.WriteByte('(')
			if _, ok := recv.Type().(*types.Interface); ok {
				// gcimporter creates abstract methods of
				// named interfaces using the interface type
				// (not the named type) as the receiver.
				// Don't print it in full.
				buf.WriteString("interface")
			} else {
				types.WriteType(buf, recv.Type(), nil)
			}
			buf.WriteByte(')')
			buf.WriteByte('.')
		} else if f.Pkg() != nil {
			writePackage(buf, f.Pkg())
		}
	}
	buf.WriteString(f.Name())
	s := buf.String()
	bufferPool.Put(buf)
	return s
}

func writePackage(buf *bytes.Buffer, pkg *types.Package) {
	if pkg == nil {
		return
	}
	s := pkg.Path()
	if s != "" {
		buf.WriteString(s)
		buf.WriteByte('.')
	}
}

type StringSliceVar []string

func (v StringSliceVar) String() string {
	return strings.Join(v, ",")
}

func (v *StringSliceVar) Set(s string) error {
	*v = StringSliceVar(strings.Split(s, ","))
	return nil
}

func (v *StringSliceVar) Get() interface{} {
	return []string(*v)
}
