// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package testutil // import "honnef.co/go/tools/lint/testutil"

import (
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/loader"
)

var lintMatch = flag.String("lint.match", "", "restrict testdata matches to this pattern")

func TestAll(t *testing.T, c lint.Checker, dir string) {
	testFiles(t, c, dir)
	testPackages(t, c, dir)
}

func testFiles(t *testing.T, c lint.Checker, dir string) {
	baseDir := filepath.Join("testdata", dir)
	fis, err := filepath.Glob(filepath.Join(baseDir, "*.go"))
	if err != nil {
		t.Fatalf("filepath.Glob: %v", err)
	}
	if len(fis) == 0 {
		t.Fatalf("no files in %v", baseDir)
	}
	rx, err := regexp.Compile(*lintMatch)
	if err != nil {
		t.Fatalf("Bad -lint.match value %q: %v", *lintMatch, err)
	}

	files := map[int][]string{}
	for _, fi := range fis {
		if !rx.MatchString(fi) {
			continue
		}
		if !strings.HasSuffix(fi, ".go") {
			continue
		}
		parts := strings.Split(fi, "_")
		v := 0
		if len(parts) > 1 && strings.HasPrefix(parts[len(parts)-1], "go1") {
			var err error
			s := parts[len(parts)-1][len("go1"):]
			s = s[:len(s)-len(".go")]
			v, err = strconv.Atoi(s)
			if err != nil {
				t.Fatalf("cannot process file name %q: %s", fi, err)
			}
		}
		files[v] = append(files[v], fi)
	}

	lprog := loader.NewProgram()
	sources := map[string][]byte{}
	var pkgs []*loader.Package
	for _, fi := range fis {
		src, err := ioutil.ReadFile(fi)
		if err != nil {
			t.Errorf("Failed reading %s: %v", fi, err)
			continue
		}
		f, err := parser.ParseFile(lprog.Fset, fi, src, parser.ParseComments)
		if err != nil {
			t.Errorf("error parsing %s: %s", fi, err)
			continue
		}
		sources[fi] = src
		pkg, err := lprog.CreateFromFiles(fi, f)
		if err != nil {
			t.Errorf("error loading %s: %s", fi, err)
			continue
		}
		pkgs = append(pkgs, pkg)
	}

	for version, fis := range files {
		lintGoVersion(t, c, version, lprog, pkgs, fis, sources)
	}
}

func testPackages(t *testing.T, c lint.Checker, dir string) {
	ctx := build.Default
	ctx.GOPATH = filepath.Join("testdata", dir)
	ctx.CgoEnabled = false
	fis, err := ioutil.ReadDir(filepath.Join(ctx.GOPATH, "src"))
	if err != nil {
		if os.IsNotExist(err) {
			// no packages to test
			return
		}
		t.Fatal("couldn't get test packages:", err)
	}

	lprog := loader.NewProgram()
	lprog.Build = &ctx
	var pkgs []*loader.Package
	var files []string
	sources := map[string][]byte{}
	for _, fi := range fis {
		pkg, err := lprog.Import(fi.Name(), ".")
		if err != nil {
			t.Fatalf("couldn't import %s: %s", fi.Name(), err)
		}
		pkgs = append(pkgs, pkg)

		groups := [][]string{
			pkg.Bpkg.GoFiles,
			pkg.Bpkg.CgoFiles,
			pkg.Bpkg.TestGoFiles,
			pkg.Bpkg.XTestGoFiles,
		}
		for _, group := range groups {
			for _, f := range group {
				p := filepath.Join(pkg.Bpkg.Dir, f)
				b, err := ioutil.ReadFile(p)
				if err != nil {
					t.Fatal("couldn't load test package:", err)
				}
				path := filepath.Join(ctx.GOPATH, "src", fi.Name(), f)
				sources[path] = b
				files = append(files, path)
			}
		}

	}

	// TODO(dh): support setting GoVersion
	lintGoVersion(t, c, 0, lprog, pkgs, files, sources)
}

func lintGoVersion(
	t *testing.T,
	c lint.Checker,
	version int,
	lprog *loader.Program,
	pkgs []*loader.Package,
	files []string,
	sources map[string][]byte,
) {
	l := &lint.Linter{Checker: c, GoVersion: version}
	res := l.Lint(lprog, pkgs)
	for _, fi := range files {
		src := sources[fi]

		ins := parseInstructions(t, fi, src)

		for _, in := range ins {
			ok := false
			for i, p := range res {
				if p.Position.Line != in.Line || p.Position.Filename != fi {
					continue
				}
				if in.Match.MatchString(p.Text) {
					// remove this problem from ps
					copy(res[i:], res[i+1:])
					res = res[:len(res)-1]

					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("Lint failed at %s:%d; /%v/ did not match", fi, in.Line, in.Match)
			}
		}
	}
	for _, p := range res {
		for _, fi := range files {
			if p.Position.Filename == fi {
				t.Errorf("Unexpected problem at %s: %v", p.Position, p.Text)
				break
			}
		}
	}
}

type instruction struct {
	Line        int            // the line number this applies to
	Match       *regexp.Regexp // what pattern to match
	Replacement string         // what the suggested replacement line should be
}

// parseInstructions parses instructions from the comments in a Go source file.
// It returns nil if none were parsed.
func parseInstructions(t *testing.T, filename string, src []byte) []instruction {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Test file %v does not parse: %v", filename, err)
	}
	var ins []instruction
	for _, cg := range f.Comments {
		ln := fset.PositionFor(cg.Pos(), false).Line
		raw := cg.Text()
		for _, line := range strings.Split(raw, "\n") {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if line == "OK" && ins == nil {
				// so our return value will be non-nil
				ins = make([]instruction, 0)
				continue
			}
			if !strings.Contains(line, "MATCH") {
				continue
			}
			rx, err := extractPattern(line)
			if err != nil {
				t.Fatalf("At %v:%d: %v", filename, ln, err)
			}
			matchLine := ln
			if i := strings.Index(line, "MATCH:"); i >= 0 {
				// This is a match for a different line.
				lns := strings.TrimPrefix(line[i:], "MATCH:")
				lns = lns[:strings.Index(lns, " ")]
				matchLine, err = strconv.Atoi(lns)
				if err != nil {
					t.Fatalf("Bad match line number %q at %v:%d: %v", lns, filename, ln, err)
				}
			}
			var repl string
			if r, ok := extractReplacement(line); ok {
				repl = r
			}
			ins = append(ins, instruction{
				Line:        matchLine,
				Match:       rx,
				Replacement: repl,
			})
		}
	}
	return ins
}

func extractPattern(line string) (*regexp.Regexp, error) {
	n := strings.Index(line, " ")
	if n == 01 {
		return nil, fmt.Errorf("malformed match instruction %q", line)
	}
	line = line[n+1:]
	var pat string
	switch line[0] {
	case '/':
		a, b := strings.Index(line, "/"), strings.LastIndex(line, "/")
		if a == -1 || a == b {
			return nil, fmt.Errorf("malformed match instruction %q", line)
		}
		pat = line[a+1 : b]
	case '"':
		a, b := strings.Index(line, `"`), strings.LastIndex(line, `"`)
		if a == -1 || a == b {
			return nil, fmt.Errorf("malformed match instruction %q", line)
		}
		pat = regexp.QuoteMeta(line[a+1 : b])
	default:
		return nil, fmt.Errorf("malformed match instruction %q", line)
	}

	rx, err := regexp.Compile(pat)
	if err != nil {
		return nil, fmt.Errorf("bad match pattern %q: %v", pat, err)
	}
	return rx, nil
}

func extractReplacement(line string) (string, bool) {
	// Look for this:  / -> `
	// (the end of a match and start of a backtick string),
	// and then the closing backtick.
	const start = "/ -> `"
	a, b := strings.Index(line, start), strings.LastIndex(line, "`")
	if a < 0 || a > b {
		return "", false
	}
	return line[a+len(start) : b], true
}
