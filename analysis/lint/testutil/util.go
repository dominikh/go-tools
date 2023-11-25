package testutil

import (
	"crypto/sha256"
	"go/build"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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

var testVersionRegexp = regexp.MustCompile(`^.+_go1(\d+)$`)

func Run(t *testing.T, a *lint.Analyzer) {
	dirs, err := filepath.Glob("testdata/src/example.com/*")
	if err != nil {
		t.Fatalf("couldn't enumerate test data: %s", err)
	}

	if len(dirs) == 0 {
		t.Fatalf("found no tests")
	}

	tests := make([]Test, 0, len(dirs))
	for _, dir := range dirs {
		// Work around Windows paths
		dir = strings.ReplaceAll(dir, `\`, `/`)
		t := Test{
			Dir: strings.TrimPrefix(dir, "testdata/src/"),
		}
		if sub := testVersionRegexp.FindStringSubmatch(dir); sub != nil {
			t.Version = "1." + sub[1]
		}
		tests = append(tests, t)
	}

	dirsByVersion := map[string][]string{}

	// Group tests by Go version so that we can run multiple tests at once, saving time and memory on type
	// checking and export data parsing.
	for _, tt := range tests {
		dirsByVersion[tt.Version] = append(dirsByVersion[tt.Version], tt.Dir)
	}

	for v, dirs := range dirsByVersion {
		actualVersion := v
		if actualVersion == "" {
			actualVersion = defaultGoVersion()
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
		res, err := r.Run(cfg, []*analysis.Analyzer{a.Analyzer}, dirs)
		if err != nil {
			t.Fatal(err)
		}

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

		for _, tt := range tests {
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

					relevantDiags := data.Diagnostics
					var relevantFacts []runner.TestFact
					for _, fact := range tdata.Facts {
						if fact.Analyzer != a.Analyzer.Name {
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
