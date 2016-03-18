package unused

// Copyright (c) 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found at
// https://developers.google.com/open-source/licenses/bsd.

import (
	"go/parser"
	"go/token"
	"io/ioutil"
	"path"
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
		checker := NewChecker(CheckAll)
		src, err := ioutil.ReadFile(path.Join(baseDir, fi.Name()))
		if err != nil {
			t.Fatalf("Failed reading %s: %v", fi.Name(), err)
		}

		ins := parseInstructions(t, fi.Name(), src)

		unused, err := checker.Check([]string{baseDir + fi.Name()})
		if err != nil {
			t.Errorf("checking %s: %v", fi.Name(), err)
			continue
		}

		for _, in := range ins {
			n := len(in.IDs)
			for _, id := range in.IDs {
				for i, u := range unused {
					if u.Position.Line != in.Line {
						continue
					}
					if id == u.Obj.Name() {
						n--
						copy(unused[i:], unused[i+1:])
						unused = unused[:len(unused)-1]
						break
					}
				}
			}
			if n != 0 {
				t.Errorf("unused failed at %s:%d; %v did not match", fi.Name(), in.Line, in.IDs)
			}
		}
		for _, u := range unused {
			t.Errorf("Unexpected unused identifier at %s:%d: %v", fi.Name(), u.Position.Line, u.Obj.Name())
		}
	}
}

type instruction struct {
	Line int // the line number this applies to
	IDs  []string
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
			if strings.Contains(line, "MATCH ") {
				ids := extractIDs(line)
				ins = append(ins, instruction{
					Line: ln,
					IDs:  ids,
				})
			}
		}
	}
	return ins
}

func extractIDs(line string) []string {
	const marker = "MATCH "
	idx := strings.Index(line, marker)
	line = line[idx+len(marker):]
	return strings.Split(line, ", ")
}
