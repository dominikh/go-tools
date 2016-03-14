package unused

// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// TODO don't do regexp checks, instead have a comma-separated list of identifiers

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestAll(t *testing.T) {
	baseDir := "testdata/"
	fis, err := ioutil.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("ioutil.ReadDir: %v", err)
	}
	if len(fis) == 0 {
		t.Fatalf("no files in %v", baseDir)
	}
	for _, fi := range fis {
		checker := NewChecker(CheckAll, false)
		src, err := ioutil.ReadFile(path.Join(baseDir, fi.Name()))
		if err != nil {
			t.Fatalf("Failed reading %s: %v", fi.Name(), err)
		}

		ins := parseInstructions(t, fi.Name(), src)
		if ins == nil {
			t.Errorf("Test file %v does not have instructions", fi.Name())
			continue
		}

		unused, err := checker.Check([]string{baseDir + fi.Name()})
		if err != nil {
			t.Errorf("checking %s: %v", fi.Name(), err)
			continue
		}

		for _, in := range ins {
			ok := false
			for i, u := range unused {
				if u.Position.Line != in.Line {
					continue
				}
				if in.Match.MatchString(u.Obj.Name()) {
					// remove this problem from ps
					copy(unused[i:], unused[i+1:])
					unused = unused[:len(unused)-1]

					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("unused failed at %s:%d; /%v/ did not match", fi.Name(), in.Line, in.Match)
			}
		}
		for _, u := range unused {
			t.Errorf("Unexpected unused identifier at %s:%d: %v", fi.Name(), u.Position.Line, u.Obj.Name())
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
				ins = append(ins, instruction{
					Line:  matchLine,
					Match: rx,
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
