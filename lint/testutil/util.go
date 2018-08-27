// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// Package testutil provides helpers for testing staticcheck.
package testutil // import "honnef.co/go/tools/lint/testutil"

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
	"honnef.co/go/tools/config"
	"honnef.co/go/tools/lint"
)

func TestAll(t *testing.T, c lint.Checker, dir string) {
	testPackages(t, c, dir)
}

func testPackages(t *testing.T, c lint.Checker, dir string) {
	gopath := filepath.Join("testdata", dir)
	gopath, err := filepath.Abs(gopath)
	if err != nil {
		t.Fatal(err)
	}
	fis, err := ioutil.ReadDir(filepath.Join(gopath, "src"))
	if err != nil {
		if os.IsNotExist(err) {
			// no packages to test
			return
		}
		t.Fatal("couldn't get test packages:", err)
	}

	var paths []string
	for _, fi := range fis {
		if strings.HasSuffix(fi.Name(), ".disabled") {
			continue
		}
		paths = append(paths, fi.Name())
	}

	conf := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: true,
		Env:   append(os.Environ(), "GOPATH="+gopath),
	}

	pkgs, err := packages.Load(conf, paths...)
	if err != nil {
		t.Error("Error loading packages:", err)
		return
	}

	versions := map[int][]*packages.Package{}
	for _, pkg := range pkgs {
		path := strings.TrimSuffix(pkg.Types.Path(), ".test")
		parts := strings.Split(path, "_")

		version := 0
		if len(parts) > 1 {
			part := parts[len(parts)-1]
			if len(part) >= 4 && strings.HasPrefix(part, "go1") {
				v, err := strconv.Atoi(part[len("go1"):])
				if err != nil {
					continue
				}
				version = v
			}
		}
		versions[version] = append(versions[version], pkg)
	}

	for version, pkgs := range versions {
		sources := map[string][]byte{}
		var files []string

		for _, pkg := range pkgs {
			files = append(files, pkg.GoFiles...)
			for _, fi := range pkg.GoFiles {
				src, err := ioutil.ReadFile(fi)
				if err != nil {
					t.Fatal(err)
				}
				sources[fi] = src
			}
		}

		sort.Strings(files)
		filesUniq := make([]string, 0, len(files))
		if len(files) < 2 {
			filesUniq = files
		} else {
			filesUniq = append(filesUniq, files[0])
			prev := files[0]
			for _, f := range files[1:] {
				if f == prev {
					continue
				}
				prev = f
				filesUniq = append(filesUniq, f)
			}
		}

		lintGoVersion(t, c, version, pkgs, filesUniq, sources)
	}
}

func lintGoVersion(
	t *testing.T,
	c lint.Checker,
	version int,
	pkgs []*packages.Package,
	files []string,
	sources map[string][]byte,
) {
	l := &lint.Linter{Checkers: []lint.Checker{c}, GoVersion: version, Config: config.Config{Checks: []string{"all"}}}
	problems := l.Lint(pkgs, nil)

	for _, fi := range files {
		src := sources[fi]

		ins := parseInstructions(t, fi, src)

		for _, in := range ins {
			ok := false
			for i, p := range problems {
				if p.Position.Line != in.Line || p.Position.Filename != fi {
					continue
				}
				if in.Match.MatchString(p.Text) {
					// remove this problem from ps
					copy(problems[i:], problems[i+1:])
					problems = problems[:len(problems)-1]

					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("Lint failed at %s:%d; /%v/ did not match", fi, in.Line, in.Match)
			}
		}
	}
	for _, p := range problems {
		t.Errorf("Unexpected problem at %s: %v", p.Position, p.Text)
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
