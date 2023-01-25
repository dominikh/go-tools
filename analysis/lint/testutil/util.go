package testutil

import (
	"crypto/sha256"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"testing"

	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/go/buildid"
	"honnef.co/go/tools/lintcmd/cache"
	"honnef.co/go/tools/lintcmd/runner"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

type Test struct {
	Dir     string
	Version string
}

func computeSalt() ([]byte, error) {
	p, err := os.Executable()
	if err != nil {
		return nil, err
	}

	if id, err := buildid.ReadFile(p); err == nil {
		return []byte(id), nil
	} else {
		// For some reason we couldn't read the build id from the executable.
		// Fall back to hashing the entire executable.
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return nil, err
		}
		return h.Sum(nil), nil
	}
}

func defaultGoVersion() string {
	tags := build.Default.ReleaseTags
	v := tags[len(tags)-1][2:]
	return v
}

func Run(t *testing.T, analyzers []*lint.Analyzer, tests map[string][]Test) {
	analyzersByName := map[string]*lint.Analyzer{}
	for _, a := range analyzers {
		analyzersByName[a.Analyzer.Name] = a
	}

	analyzersByVersion := map[string]map[*lint.Analyzer]struct{}{}
	dirsByVersion := map[string][]string{}

	for analyzerName, ttt := range tests {
		for _, tt := range ttt {
			m := analyzersByVersion[tt.Version]
			if m == nil {
				m = map[*lint.Analyzer]struct{}{}
				analyzersByVersion[tt.Version] = m
			}

			analyzer, ok := analyzersByName[analyzerName]
			if !ok {
				t.Errorf("found tests for analyzer %q, but no such analyzer exists", analyzerName)
				continue
			}
			m[analyzer] = struct{}{}

			dirsByVersion[tt.Version] = append(dirsByVersion[tt.Version], tt.Dir)
		}
	}

	for v, asm := range analyzersByVersion {
		dirs := dirsByVersion[v]

		actualVersion := v
		if actualVersion == "" {
			actualVersion = defaultGoVersion()
		}
		as := make([]*analysis.Analyzer, 0, len(asm))
		for a := range asm {
			as = append(as, a.Analyzer)
			if err := a.Analyzer.Flags.Lookup("go").Value.Set(actualVersion); err != nil {
				t.Fatal(err)
			}
		}

		c, err := cache.Open(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		salt, err := computeSalt()
		if err != nil {
			t.Fatal(err)
		}
		c.SetSalt(salt)
		r, err := runner.New(config.Config{}, c)
		if err != nil {
			t.Fatal(err)
		}
		r.GoVersion = actualVersion
		r.TestMode = true

		testdata, err := filepath.Abs("testdata")
		if err != nil {
			t.Fatal(err)
		}
		cfg := &packages.Config{
			Tests: true,
			Env:   append(os.Environ(), "GOPATH="+testdata, "GO111MODULE=off", "GOPROXY=off"),
		}
		if len(dirs) == 0 {
			t.Fatal("no directories for version", v)
		}
		res, err := r.Run(cfg, as, dirs)
		if err != nil {
			t.Fatal(err)
		}

		// Each result in res contains all diagnostics and facts for all checked packages for all checked analyzers.
		// For each package, we only care about the diagnostics and facts reported by a single analyzer.

		// resultByPath maps from import path to results
		resultByPath := map[string][]runner.Result{}
		failed := false
		for _, r := range res {
			if r.Failed {
				failed = true
				if len(r.Errors) > 0 {
					t.Fatalf("failed checking %s: %v", r.Package.PkgPath, r.Errors)
				}
			}
			// r.Package.PkgPath is not unique. The same path can refer to a package and a package plus its
			// (non-external) tests.
			resultByPath[r.Package.PkgPath] = append(resultByPath[r.Package.PkgPath], r)
		}

		if failed {
			t.Fatal("failed processing package, but got no errors")
		}

		for a, ttt := range tests {
			for _, tt := range ttt {
				if tt.Version != v {
					continue
				}
				any := false
				for _, suffix := range []string{"", ".test", "_test"} {
					dir := tt.Dir + suffix
					rr, ok := resultByPath[dir]
					if !ok {
						continue
					}
					any = true
					// Remove this result. We later check that there remain no tests we haven't checked.
					delete(resultByPath, dir)

					for _, r := range rr {
						data, err := r.Load()
						if err != nil {
							t.Fatal(err)
						}
						tdata, err := r.LoadTest()
						if err != nil {
							t.Fatal(err)
						}

						// Select those diagnostics made by the analyzer we're currently checking
						var relevantDiags []runner.Diagnostic
						for _, diag := range data.Diagnostics {
							// FIXME(dh): Category might not match analyzer names. it does for Staticcheck, for now
							if diag.Category != a {
								continue
							}
							relevantDiags = append(relevantDiags, diag)
						}

						var relevantFacts []runner.TestFact
						for _, fact := range tdata.Facts {
							if fact.Analyzer != a {
								continue
							}
							relevantFacts = append(relevantFacts, fact)
						}

						Check(t, testdata, tdata.Files, relevantDiags, relevantFacts)
						CheckSuggestedFixes(t, relevantDiags)
					}
				}
				if !any {
					t.Errorf("no result for directory %s", tt.Dir)
				}
			}
		}
		for key, rr := range resultByPath {
			for _, r := range rr {
				data, err := r.Load()
				if err != nil {
					t.Fatal(err)
				}
				if len(data.Diagnostics) != 0 {
					t.Errorf("unexpected diagnostics in package %s", key)
				}
			}
		}
	}
}
