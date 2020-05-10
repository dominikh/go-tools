// Package lint provides the foundation for tools like staticcheck
package lint // import "honnef.co/go/tools/lint"

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"honnef.co/go/tools/config"
	"honnef.co/go/tools/runner"
	"honnef.co/go/tools/unused"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

type Documentation struct {
	Title      string
	Text       string
	Since      string
	NonDefault bool
	Options    []string
}

func (doc *Documentation) String() string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "%s\n\n", doc.Title)
	if doc.Text != "" {
		fmt.Fprintf(b, "%s\n\n", doc.Text)
	}
	fmt.Fprint(b, "Available since\n    ")
	if doc.Since == "" {
		fmt.Fprint(b, "unreleased")
	} else {
		fmt.Fprintf(b, "%s", doc.Since)
	}
	if doc.NonDefault {
		fmt.Fprint(b, ", non-default")
	}
	fmt.Fprint(b, "\n")
	if len(doc.Options) > 0 {
		fmt.Fprintf(b, "\nOptions\n")
		for _, opt := range doc.Options {
			fmt.Fprintf(b, "    %s", opt)
		}
		fmt.Fprint(b, "\n")
	}
	return b.String()
}

type ignore interface {
	Match(p Problem) bool
}

type lineIgnore struct {
	File    string
	Line    int
	Checks  []string
	Matched bool
	Pos     token.Position
}

func (li *lineIgnore) Match(p Problem) bool {
	pos := p.Position
	if pos.Filename != li.File || pos.Line != li.Line {
		return false
	}
	for _, c := range li.Checks {
		if m, _ := filepath.Match(c, p.Category); m {
			li.Matched = true
			return true
		}
	}
	return false
}

func (li *lineIgnore) String() string {
	matched := "not matched"
	if li.Matched {
		matched = "matched"
	}
	return fmt.Sprintf("%s:%d %s (%s)", li.File, li.Line, strings.Join(li.Checks, ", "), matched)
}

type fileIgnore struct {
	File   string
	Checks []string
}

func (fi *fileIgnore) Match(p Problem) bool {
	if p.Position.Filename != fi.File {
		return false
	}
	for _, c := range fi.Checks {
		if m, _ := filepath.Match(c, p.Category); m {
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
	runner.Diagnostic
	Severity Severity
}

func (p Problem) equal(o Problem) bool {
	return p.Position == o.Position &&
		p.End == o.End &&
		p.Message == o.Message &&
		p.Category == o.Category &&
		p.Severity == o.Severity
}

func (p *Problem) String() string {
	return fmt.Sprintf("%s (%s)", p.Message, p.Category)
}

// A Linter lints Go source code.
type Linter struct {
	Checkers []*analysis.Analyzer
	Config   config.Config
	Runner   *runner.Runner
}

func failed(res runner.Result) []Problem {
	var problems []Problem

	for _, e := range res.Errors {
		switch e := e.(type) {
		case packages.Error:
			msg := e.Msg
			if len(msg) != 0 && msg[0] == '\n' {
				// TODO(dh): See https://github.com/golang/go/issues/32363
				msg = msg[1:]
			}

			var posn token.Position
			if e.Pos == "" {
				// Under certain conditions (malformed package
				// declarations, multiple packages in the same
				// directory), go list emits an error on stderr
				// instead of JSON. Those errors do not have
				// associated position information in
				// go/packages.Error, even though the output on
				// stderr may contain it.
				if p, n, err := parsePos(msg); err == nil {
					if abs, err := filepath.Abs(p.Filename); err == nil {
						p.Filename = abs
					}
					posn = p
					msg = msg[n+2:]
				}
			} else {
				var err error
				posn, _, err = parsePos(e.Pos)
				if err != nil {
					panic(fmt.Sprintf("internal error: %s", e))
				}
			}
			p := Problem{
				Diagnostic: runner.Diagnostic{
					Position: posn,
					Message:  msg,
					Category: "compile",
				},
				Severity: Error,
			}
			problems = append(problems, p)
		case error:
			p := Problem{
				Diagnostic: runner.Diagnostic{
					Position: token.Position{},
					Message:  e.Error(),
					Category: "compile",
				},
				Severity: Error,
			}
			problems = append(problems, p)
		}
	}

	return problems
}

type unusedKey struct {
	pkgPath string
	base    string
	line    int
	name    string
}

type unusedPair struct {
	key unusedKey
	obj unused.SerializedObject
}

func success(allowedChecks map[string]bool, res runner.Result) ([]Problem, unused.SerializedResult, error) {
	diags, err := res.Diagnostics()
	if err != nil {
		return nil, unused.SerializedResult{}, err
	}

	var problems []Problem

	for _, diag := range diags {
		if !allowedChecks[diag.Category] {
			continue
		}
		problems = append(problems, Problem{Diagnostic: diag})
	}

	u, err := res.Unused()
	return problems, u, err
}

func filterIgnored(problems []Problem, res runner.Result, allowedAnalyzers map[string]bool) ([]Problem, error) {
	couldveMatched := func(ig *lineIgnore) bool {
		for _, c := range ig.Checks {
			if c == "U1000" {
				// We never want to flag ignores for U1000,
				// because U1000 isn't local to a single
				// package. For example, an identifier may
				// only be used by tests, in which case an
				// ignore would only fire when not analyzing
				// tests. To avoid spurious "useless ignore"
				// warnings, just never flag U1000.
				return false
			}

			// Even though the runner always runs all analyzers, we
			// still only flag unmatched ignores for the set of
			// analyzers the user has expressed interest in. That way,
			// `staticcheck -checks=SA1000` won't complain about an
			// unmatched ignore for an unrelated check.
			if allowedAnalyzers[c] {
				return true
			}
		}

		return false
	}

	dirs, err := res.Directives()
	if err != nil {
		return nil, err
	}

	ignores, moreProblems := parseDirectives(dirs)

	for _, ig := range ignores {
		for i := range problems {
			p := &problems[i]
			if ig.Match(*p) {
				p.Severity = Ignored
			}
		}

		if ig, ok := ig.(*lineIgnore); ok && !ig.Matched && couldveMatched(ig) {
			p := Problem{
				Diagnostic: runner.Diagnostic{
					Position: ig.Pos,
					Message:  "this linter directive didn't match anything; should it be removed?",
					Category: "",
				},
			}
			moreProblems = append(moreProblems, p)
		}
	}

	return append(problems, moreProblems...), nil
}

func NewLinter(cfg config.Config) (*Linter, error) {
	r, err := runner.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Linter{
		Config: cfg,
		Runner: r,
	}, nil
}

func (l *Linter) SetGoVersion(n int) {
	l.Runner.GoVersion = n
}

func (l *Linter) Lint(cfg *packages.Config, patterns []string) ([]Problem, error) {
	results, err := l.Runner.Run(cfg, l.Checkers, patterns)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 && err == nil {
		// TODO(dh): emulate Go's behavior more closely once we have
		// access to go list's Match field.
		fmt.Fprintf(os.Stderr, "warning: %q matched no packages\n", patterns)
	}

	analyzerNames := make([]string, len(l.Checkers))
	for i, a := range l.Checkers {
		analyzerNames[i] = a.Name
	}

	var problems []Problem
	used := map[unusedKey]bool{}
	var unuseds []unusedPair
	for _, res := range results {
		if len(res.Errors) > 0 && !res.Failed {
			panic("package has errors but isn't marked as failed")
		}
		if res.Failed {
			problems = append(problems, failed(res)...)
		} else {
			if !res.Initial {
				continue
			}

			allowedAnalyzers := FilterAnalyzerNames(analyzerNames, res.Config.Checks)
			ps, u, err := success(allowedAnalyzers, res)
			if err != nil {
				return nil, err
			}
			filtered, err := filterIgnored(ps, res, allowedAnalyzers)
			if err != nil {
				return nil, err
			}
			problems = append(problems, filtered...)

			for _, obj := range u.Used {
				// FIXME(dh): pick the object whose filename does not include $GOROOT
				key := unusedKey{
					pkgPath: res.Package.PkgPath,
					base:    filepath.Base(obj.Position.Filename),
					line:    obj.Position.Line,
					name:    obj.Name,
				}
				used[key] = true
			}

			if allowedAnalyzers["U1000"] {
				for _, obj := range u.Unused {
					key := unusedKey{
						pkgPath: res.Package.PkgPath,
						base:    filepath.Base(obj.Position.Filename),
						line:    obj.Position.Line,
						name:    obj.Name,
					}
					unuseds = append(unuseds, unusedPair{key, obj})
					if _, ok := used[key]; !ok {
						used[key] = false
					}
				}
			}
		}
	}

	for _, uo := range unuseds {
		if used[uo.key] {
			continue
		}
		if uo.obj.InGenerated {
			continue
		}
		problems = append(problems, Problem{
			Diagnostic: runner.Diagnostic{
				Position: uo.obj.DisplayPosition,
				Message:  fmt.Sprintf("%s %s is unused", uo.obj.Kind, uo.obj.Name),
				Category: "U1000",
			},
		})
	}

	if len(problems) == 0 {
		return nil, nil
	}

	sort.Slice(problems, func(i, j int) bool {
		pi := problems[i].Position
		pj := problems[j].Position

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
		if !problems[i].equal(p) {
			out = append(out, p)
		}
	}
	return out, nil
}

func FilterAnalyzerNames(analyzers []string, checks []string) map[string]bool {
	allowedChecks := map[string]bool{}

	for _, check := range checks {
		b := true
		if len(check) > 1 && check[0] == '-' {
			b = false
			check = check[1:]
		}
		if check == "*" || check == "all" {
			// Match all
			for _, c := range analyzers {
				allowedChecks[c] = b
			}
		} else if strings.HasSuffix(check, "*") {
			// Glob
			prefix := check[:len(check)-1]
			isCat := strings.IndexFunc(prefix, func(r rune) bool { return unicode.IsNumber(r) }) == -1

			for _, a := range analyzers {
				idx := strings.IndexFunc(a, func(r rune) bool { return unicode.IsNumber(r) })
				if isCat {
					// Glob is S*, which should match S1000 but not SA1000
					cat := a[:idx]
					if prefix == cat {
						allowedChecks[a] = b
					}
				} else {
					// Glob is S1*
					if strings.HasPrefix(a, prefix) {
						allowedChecks[a] = b
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

var posRe = regexp.MustCompile(`^(.+?):(\d+)(?::(\d+)?)?`)

func parsePos(pos string) (token.Position, int, error) {
	if pos == "-" || pos == "" {
		return token.Position{}, 0, nil
	}
	parts := posRe.FindStringSubmatch(pos)
	if parts == nil {
		return token.Position{}, 0, fmt.Errorf("malformed position %q", pos)
	}
	file := parts[1]
	line, _ := strconv.Atoi(parts[2])
	col, _ := strconv.Atoi(parts[3])
	return token.Position{
		Filename: file,
		Line:     line,
		Column:   col,
	}, len(parts[0]), nil
}
