// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file is a modified copy of x/tools/go/analysis/analysistest/analysistest.go

package testutil

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"honnef.co/go/tools/internal/diff/myers"
	"honnef.co/go/tools/lintcmd/runner"

	"golang.org/x/tools/go/expect"
	"golang.org/x/tools/txtar"
)

func CheckSuggestedFixes(t *testing.T, diagnostics []runner.Diagnostic) {
	// Process each result (package) separately, matching up the suggested
	// fixes into a diff, which we will compare to the .golden file.  We have
	// to do this per-result in case a file appears in two packages, such as in
	// packages with tests, where mypkg/a.go will appear in both mypkg and
	// mypkg.test.  In that case, the analyzer may suggest the same set of
	// changes to a.go for each package.  If we merge all the results, those
	// changes get doubly applied, which will cause conflicts or mismatches.
	// Validating the results separately means as long as the two analyses
	// don't produce conflicting suggestions for a single file, everything
	// should match up.
	// file -> message -> edits
	fileEdits := make(map[string]map[string][]runner.TextEdit)
	fileContents := make(map[string][]byte)

	// Validate edits, prepare the fileEdits map and read the file contents.
	for _, diag := range diagnostics {
		for _, sf := range diag.SuggestedFixes {
			for _, edit := range sf.TextEdits {
				// Validate the edit.
				if edit.Position.Offset > edit.End.Offset {
					t.Errorf(
						"diagnostic for analysis %v contains Suggested Fix with malformed edit: pos (%v) > end (%v)",
						diag.Category, edit.Position.Offset, edit.End.Offset)
					continue
				}
				if edit.Position.Filename != edit.End.Filename {
					t.Errorf(
						"diagnostic for analysis %v contains Suggested Fix with malformed edit spanning files %v and %v",
						diag.Category, edit.Position.Filename, edit.End.Filename)
					continue
				}
				if _, ok := fileContents[edit.Position.Filename]; !ok {
					contents, err := os.ReadFile(edit.Position.Filename)
					if err != nil {
						t.Errorf("error reading %s: %v", edit.Position.Filename, err)
					}
					fileContents[edit.Position.Filename] = contents
				}

				if _, ok := fileEdits[edit.Position.Filename]; !ok {
					fileEdits[edit.Position.Filename] = make(map[string][]runner.TextEdit)
				}
				fileEdits[edit.Position.Filename][sf.Message] = append(fileEdits[edit.Position.Filename][sf.Message], edit)
			}
		}
	}

	for file, fixes := range fileEdits {
		// Get the original file contents.
		orig, ok := fileContents[file]
		if !ok {
			t.Errorf("could not find file contents for %s", file)
			continue
		}

		// Get the golden file and read the contents.
		ar, err := txtar.ParseFile(file + ".golden")
		if err != nil {
			t.Errorf("error reading %s.golden: %v", file, err)
			continue
		}

		if len(ar.Files) > 0 {
			// one virtual file per kind of suggested fix

			if len(ar.Comment) != 0 {
				// we allow either just the comment, or just virtual
				// files, not both. it is not clear how "both" should
				// behave.
				t.Errorf("%s.golden has leading comment; we don't know what to do with it", file)
				continue
			}

			var sfs []string
			for sf := range fixes {
				sfs = append(sfs, sf)
			}
			sort.Slice(sfs, func(i, j int) bool {
				return sfs[i] < sfs[j]
			})
			for _, sf := range sfs {
				edits := fixes[sf]
				found := false
				for _, vf := range ar.Files {
					if vf.Name == sf {
						found = true
						out := applyEdits(orig, edits)
						// the file may contain multiple trailing
						// newlines if the user places empty lines
						// between files in the archive. normalize
						// this to a single newline.
						want := string(bytes.TrimRight(vf.Data, "\n")) + "\n"
						formatted, err := format.Source([]byte(out))
						if err != nil {
							t.Errorf("%s: error formatting edited source: %v\n%s", file, err, out)
							continue
						}
						if want != string(formatted) {
							d := myers.ComputeEdits(want, string(formatted))
							diff := ""
							for _, op := range d {
								diff += op.String()
							}
							t.Errorf("suggested fixes failed for %s[%s]:\n%s", file, sf, diff)
						}
						break
					}
				}
				if !found {
					t.Errorf("no section for suggested fix %q in %s.golden", sf, file)
				}
			}
			for _, vf := range ar.Files {
				if _, ok := fixes[vf.Name]; !ok {
					t.Errorf("%s.golden has section for suggested fix %q, but we didn't produce any fix by that name", file, vf.Name)
				}
			}
		} else {
			// all suggested fixes are represented by a single file

			var catchallEdits []runner.TextEdit
			for _, edits := range fixes {
				catchallEdits = append(catchallEdits, edits...)
			}

			out := applyEdits(orig, catchallEdits)
			want := string(ar.Comment)

			formatted, err := format.Source([]byte(out))
			if err != nil {
				t.Errorf("%s: error formatting resulting source: %v\n%s", file, err, out)
				continue
			}
			if want != string(formatted) {
				d := myers.ComputeEdits(want, string(formatted))
				diff := ""
				for _, op := range d {
					diff += op.String()
				}
				t.Errorf("suggested fixes failed for %s:\n%s", file, diff)
			}
		}
	}
}

func Check(t *testing.T, gopath string, files []string, diagnostics []runner.Diagnostic, facts []runner.TestFact) {
	relativePath := func(path string) string {
		cwd, err := os.Getwd()
		if err != nil {
			return path
		}
		rel, err := filepath.Rel(cwd, path)
		if err != nil {
			return path
		}
		return rel
	}

	type key struct {
		file string
		line int
	}

	// the 'files' argument contains a list of all files that were part of the tested package
	want := make(map[key][]*expect.Note)

	fset := token.NewFileSet()
	seen := map[string]struct{}{}
	for _, file := range files {
		seen[file] = struct{}{}

		notes, err := expect.Parse(fset, file, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, note := range notes {
			k := key{
				file: file,
				line: fset.PositionFor(note.Pos, false).Line,
			}
			want[k] = append(want[k], note)
		}
	}

	for _, diag := range diagnostics {
		file := diag.Position.Filename
		if _, ok := seen[file]; !ok {
			t.Errorf("got diagnostic in file %q, but that file isn't part of the checked package", relativePath(file))
			return
		}
	}

	check := func(posn token.Position, message string, kind string, argIdx int, identifier string) {
		k := key{posn.Filename, posn.Line}
		expects := want[k]
		var unmatched []string
		for i, exp := range expects {
			if exp.Name == kind {
				if kind == "fact" && exp.Args[0] != expect.Identifier(identifier) {
					continue
				}
				matched := false
				switch arg := exp.Args[argIdx].(type) {
				case string:
					matched = strings.Contains(message, arg)
				case *regexp.Regexp:
					matched = arg.MatchString(message)
				default:
					t.Fatalf("unexpected argument type %T", arg)
				}
				if matched {
					// matched: remove the expectation.
					expects[i] = expects[len(expects)-1]
					expects = expects[:len(expects)-1]
					want[k] = expects
					return
				}
				unmatched = append(unmatched, fmt.Sprintf("%q", exp.Args[argIdx]))
			}
		}
		if unmatched == nil {
			posn.Filename = relativePath(posn.Filename)
			t.Errorf("%v: unexpected diag: %v", posn, message)
		} else {
			posn.Filename = relativePath(posn.Filename)
			t.Errorf("%v: diag %q does not match pattern %s",
				posn, message, strings.Join(unmatched, " or "))
		}
	}

	checkDiag := func(posn token.Position, message string) {
		check(posn, message, "diag", 0, "")
	}

	checkFact := func(posn token.Position, name, message string) {
		check(posn, message, "fact", 1, name)
	}

	// Check the diagnostics match expectations.
	for _, f := range diagnostics {
		// TODO(matloob): Support ranges in analysistest.
		posn := f.Position
		checkDiag(posn, f.Message)
	}

	// Check the facts match expectations.
	for _, fact := range facts {
		name := fact.ObjectName
		posn := fact.Position
		if name == "" {
			name = "package"
			posn.Line = 1
		}

		checkFact(posn, name, fact.FactString)
	}

	// Reject surplus expectations.
	//
	// Sometimes an Analyzer reports two similar diagnostics on a
	// line with only one expectation. The reader may be confused by
	// the error message.
	// TODO(adonovan): print a better error:
	// "got 2 diagnostics here; each one needs its own expectation".
	var surplus []string
	for key, expects := range want {
		for _, exp := range expects {
			surplus = append(surplus, fmt.Sprintf("%s:%d: no %s was reported matching %q", relativePath(key.file), key.line, exp.Name, exp.Args))
		}
	}
	sort.Strings(surplus)
	for _, err := range surplus {
		t.Errorf("%s", err)
	}
}

func applyEdits(src []byte, edits []runner.TextEdit) []byte {
	// This function isn't efficient, but it doesn't have to be.

	edits = append([]runner.TextEdit(nil), edits...)
	sort.Slice(edits, func(i, j int) bool {
		if edits[i].Position.Offset < edits[j].Position.Offset {
			return true
		}
		if edits[i].Position.Offset == edits[j].Position.Offset {
			return edits[i].End.Offset < edits[j].End.Offset
		}
		return false
	})

	out := append([]byte(nil), src...)
	offset := 0
	for _, edit := range edits {
		start := edit.Position.Offset + offset
		end := edit.End.Offset + offset
		if edit.End == (token.Position{}) {
			end = -1
		}
		if len(edit.NewText) == 0 {
			// pure deletion
			copy(out[start:], out[end:])
			out = out[:len(out)-(end-start)]
			offset -= end - start
		} else if end == -1 || end == start {
			// pure insertion
			tmp := make([]byte, len(out)+len(edit.NewText))
			copy(tmp, out[:start])
			copy(tmp[start:], edit.NewText)
			copy(tmp[start+len(edit.NewText):], out[start:])
			offset += len(edit.NewText)
			out = tmp
		} else if end-start == len(edit.NewText) {
			// exact replacement
			copy(out[start:], edit.NewText)
		} else if end-start < len(edit.NewText) {
			// replace with longer string
			growth := len(edit.NewText) - (end - start)
			tmp := make([]byte, len(out)+growth)
			copy(tmp, out[:start])
			copy(tmp[start:], edit.NewText)
			copy(tmp[start+len(edit.NewText):], out[end:])
			offset += growth
			out = tmp
		} else if end-start > len(edit.NewText) {
			// replace with shorter string
			shrinkage := (end - start) - len(edit.NewText)

			copy(out[start:], edit.NewText)
			copy(out[start+len(edit.NewText):], out[end:])
			out = out[:len(out)-shrinkage]
			offset -= shrinkage
		}
	}

	// Debug code
	if false {
		fmt.Println("input:")
		fmt.Println(string(src))
		fmt.Println()
		fmt.Println("edits:")
		for _, edit := range edits {
			fmt.Printf("%d:%d - %d:%d <- %q\n", edit.Position.Line, edit.Position.Column, edit.End.Line, edit.End.Column, edit.NewText)
		}
		fmt.Println("output:")
		fmt.Println(string(out))
		panic("")
	}

	return out
}
