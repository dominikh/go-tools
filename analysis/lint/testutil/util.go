package testutil

import (
	"crypto/sha256"
	"go/build"
	"go/version"
	"io"
	"os"
	"path/filepath"
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

func Run(t *testing.T, a *lint.Analyzer) {
	dirs, err := filepath.Glob("testdata/*")
	if err != nil {
		t.Fatalf("couldn't enumerate test data: %s", err)
	}

	if len(dirs) == 0 {
		t.Fatalf("found no tests")
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

	tags := build.Default.ReleaseTags
	maxVersion := tags[len(tags)-1]
	for _, dir := range dirs {
		vers := filepath.Base(dir)
		t.Run(vers, func(t *testing.T) {
			if !version.IsValid(vers) {
				t.Fatalf("%q is not a valid Go version", dir)
			}
			if version.Compare(vers, maxVersion) == 1 {
				t.Skipf("%s is newer than our Go version (%s), skipping", vers, maxVersion)
			}
			r, err := runner.New(config.Config{}, c)
			if err != nil {
				t.Fatal(err)
			}
			r.TestMode = true

			testdata, err := filepath.Abs("testdata")
			if err != nil {
				t.Fatal(err)
			}
			cfg := &packages.Config{
				Dir:   dir,
				Tests: true,
				Env:   append(os.Environ(), "GOPROXY=off", "GOFLAGS=-mod=vendor", "GO111MODULE="),
				Overlay: map[string][]byte{
					"go.mod": []byte("module example.com\ngo " + strings.TrimPrefix(vers, "go")),
				},
			}
			res, err := r.Run(cfg, []*analysis.Analyzer{a.Analyzer}, []string{"./..."})
			if err != nil {
				t.Fatal(err)
			}

			if len(res) == 0 {
				t.Fatalf("got no results for %s/...", dir)
			}

			for _, r := range res {
				if r.Failed {
					if len(r.Errors) > 0 {
						sb := strings.Builder{}
						for _, err := range r.Errors {
							sb.WriteString(err.Error())
							sb.WriteString("\n")
						}
						t.Fatalf("failed checking %s:\n%s", r.Package.PkgPath, sb.String())
					} else {
						t.Fatalf("failed processing package %s, but got no errors", r.Package.PkgPath)
					}
				}
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
		})
	}
}
