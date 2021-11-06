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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"text/scanner"

	"honnef.co/go/tools/internal/diff/myers"
	"honnef.co/go/tools/lintcmd/runner"

	"golang.org/x/tools/txtar"
)

type expectation struct {
	kind string // either "fact" or "diagnostic"
	name string // name of object to which fact belongs, or "package" ("fact" only)
	rx   *regexp.Regexp
}

func (ex expectation) String() string {
	return fmt.Sprintf("%s %s:%q", ex.kind, ex.name, ex.rx) // for debugging
}

// sanitize removes the GOPATH portion of the filename,
// typically a gnarly /tmp directory, and returns the rest.
func sanitize(gopath, filename string) string {
	prefix := gopath + string(os.PathSeparator) + "src" + string(os.PathSeparator)
	return filepath.ToSlash(strings.TrimPrefix(filename, prefix))
}

// parseExpectations parses the content of a "// want ..." comment
// and returns the expectations, a mixture of diagnostics ("rx") and
// facts (name:"rx").
func parseExpectations(text string) (lineDelta int, expects []expectation, err error) {
	var scanErr string
	sc := new(scanner.Scanner).Init(strings.NewReader(text))
	sc.Error = func(s *scanner.Scanner, msg string) {
		scanErr = msg // e.g. bad string escape
	}
	sc.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanInts

	scanRegexp := func(tok rune) (*regexp.Regexp, error) {
		if tok != scanner.String && tok != scanner.RawString {
			return nil, fmt.Errorf("got %s, want regular expression",
				scanner.TokenString(tok))
		}
		pattern, _ := strconv.Unquote(sc.TokenText()) // can't fail
		return regexp.Compile(pattern)
	}

	for {
		tok := sc.Scan()
		switch tok {
		case '+':
			tok = sc.Scan()
			if tok != scanner.Int {
				return 0, nil, fmt.Errorf("got +%s, want +Int", scanner.TokenString(tok))
			}
			lineDelta, _ = strconv.Atoi(sc.TokenText())
		case scanner.String, scanner.RawString:
			rx, err := scanRegexp(tok)
			if err != nil {
				return 0, nil, err
			}
			expects = append(expects, expectation{"diagnostic", "", rx})

		case scanner.Ident:
			name := sc.TokenText()
			tok = sc.Scan()
			if tok != ':' {
				return 0, nil, fmt.Errorf("got %s after %s, want ':'",
					scanner.TokenString(tok), name)
			}
			tok = sc.Scan()
			rx, err := scanRegexp(tok)
			if err != nil {
				return 0, nil, err
			}
			expects = append(expects, expectation{"fact", name, rx})

		case scanner.EOF:
			if scanErr != "" {
				return 0, nil, fmt.Errorf("%s", scanErr)
			}
			return lineDelta, expects, nil

		default:
			return 0, nil, fmt.Errorf("unexpected %s", scanner.TokenString(tok))
		}
	}
}

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
					contents, err := ioutil.ReadFile(edit.Position.Filename)
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

func Check(t *testing.T, gopath string, diagnostics []runner.Diagnostic, wants []runner.Want, facts []runner.TestFact) {
	type key struct {
		file string
		line int
	}

	want := make(map[key][]expectation)

	// processComment parses expectations out of comments.
	processComment := func(filename string, linenum int, text string) {
		text = strings.TrimSpace(text)

		// Any comment starting with "want" is treated
		// as an expectation, even without following whitespace.
		if rest := strings.TrimPrefix(text, "want"); rest != text {
			lineDelta, expects, err := parseExpectations(rest)
			if err != nil {
				t.Errorf("%s:%d: in 'want' comment: %s", filename, linenum, err)
				return
			}
			if expects != nil {
				want[key{filename, linenum + lineDelta}] = expects
			}
		}
	}

	for _, want := range wants {
		filename := sanitize(gopath, want.Position.Filename)
		processComment(filename, want.Position.Line, want.Comment)
	}

	checkMessage := func(posn token.Position, kind, name, message string) {
		posn.Filename = sanitize(gopath, posn.Filename)
		k := key{posn.Filename, posn.Line}
		expects := want[k]
		var unmatched []string
		for i, exp := range expects {
			if exp.kind == kind && exp.name == name {
				if exp.rx.MatchString(message) {
					// matched: remove the expectation.
					expects[i] = expects[len(expects)-1]
					expects = expects[:len(expects)-1]
					want[k] = expects
					return
				}
				unmatched = append(unmatched, fmt.Sprintf("%q", exp.rx))
			}
		}
		if unmatched == nil {
			t.Errorf("%v: unexpected %s: %v", posn, kind, message)
		} else {
			t.Errorf("%v: %s %q does not match pattern %s",
				posn, kind, message, strings.Join(unmatched, " or "))
		}
	}

	// Check the diagnostics match expectations.
	for _, f := range diagnostics {
		// TODO(matloob): Support ranges in analysistest.
		posn := f.Position
		checkMessage(posn, "diagnostic", "", f.Message)
	}

	// Check the facts match expectations.
	for _, fact := range facts {
		name := fact.ObjectName
		posn := fact.Position
		if name == "" {
			name = "package"
			posn.Line = 1
		}

		checkMessage(posn, "fact", name, fact.FactString)
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
			err := fmt.Sprintf("%s:%d: no %s was reported matching %q", key.file, key.line, exp.kind, exp.rx)
			surplus = append(surplus, err)
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
