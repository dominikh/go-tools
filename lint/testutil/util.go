// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package testutil // import "honnef.co/go/lint/testutil"

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"honnef.co/go/lint"

	"golang.org/x/tools/go/loader"
)

var lintMatch = flag.String("lint.match", "", "restrict testdata matches to this pattern")

func TestAll(t *testing.T, c lint.Checker, dir string) {
	l := &lint.Linter{Checker: c}
	rx, err := regexp.Compile(*lintMatch)
	if err != nil {
		t.Fatalf("Bad -lint.match value %q: %v", *lintMatch, err)
	}

	baseDir := filepath.Join("testdata", dir)
	fis, err := ioutil.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("ioutil.ReadDir: %v", err)
	}
	if len(fis) == 0 {
		t.Fatalf("no files in %v", baseDir)
	}

	conf := &loader.Config{
		ParserMode: parser.ParseComments,
	}
	files := map[string][]byte{}
	for _, fi := range fis {
		if !rx.MatchString(fi.Name()) {
			continue
		}
		if !strings.HasSuffix(fi.Name(), ".go") {
			continue
		}
		filename := path.Join(baseDir, fi.Name())
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Errorf("Failed reading %s: %v", fi.Name(), err)
			continue
		}
		f, err := conf.ParseFile(filename, src)
		if err != nil {
			t.Errorf("error parsing %s: %s", filename, err)
			continue
		}
		files[fi.Name()] = src
		conf.CreateFromFiles(fi.Name(), f)
	}

	lprog, err := conf.Load()
	if err != nil {
		t.Fatalf("error loading program: %s", err)
	}
	res := l.Lint(lprog)
	for name, src := range files {
		ins := parseInstructions(t, name, src)

		ps := res[name]
		for _, in := range ins {
			ok := false
			for i, p := range ps {
				if p.Position.Line != in.Line {
					continue
				}
				if in.Match.MatchString(p.Text) {
					// check replacement if we are expecting one
					if in.Replacement != "" {
						// ignore any inline comments, since that would be recursive
						r := p.ReplacementLine
						if i := strings.Index(r, " //"); i >= 0 {
							r = r[:i]
						}
						if r != in.Replacement {
							t.Errorf("Lint failed at %s:%d; got replacement %q, want %q", name, in.Line, r, in.Replacement)
						}
					}

					// remove this problem from ps
					copy(ps[i:], ps[i+1:])
					ps = ps[:len(ps)-1]

					//t.Logf("/%v/ matched at %s:%d", in.Match, fi.Name(), in.Line)
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("Lint failed at %s:%d; /%v/ did not match", name, in.Line, in.Match)
			}
		}
		for _, p := range ps {
			t.Errorf("Unexpected problem at %s:%d: %v", name, p.Position.Line, p.Text)
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
		ln := fset.Position(cg.Pos()).Line
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
			if strings.Contains(line, "MATCH") {
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
	}
	return ins
}

func extractPattern(line string) (*regexp.Regexp, error) {
	a, b := strings.Index(line, "/"), strings.LastIndex(line, "/")
	if a == -1 || a == b {
		return nil, fmt.Errorf("malformed match instruction %q", line)
	}
	pat := line[a+1 : b]
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
