package facts

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// A directive is a comment of the form '//lint:<command>
// [arguments...]'. It represents instructions to the static analysis
// tool.
type Directive struct {
	Command   string
	Arguments []string
	Directive *ast.Comment
	Node      ast.Node
}

type SerializedDirective struct {
	Command   string
	Arguments []string
	// The position of the comment
	DirectivePosition token.Position
	// The position of the node that the comment is attached to
	NodePosition token.Position
}

func parseDirective(s string) (cmd string, args []string) {
	if !strings.HasPrefix(s, "//lint:") {
		return "", nil
	}
	s = strings.TrimPrefix(s, "//lint:")
	fields := strings.Split(s, " ")
	return fields[0], fields[1:]
}

func directives(pass *analysis.Pass) (interface{}, error) {
	return ParseDirectives(pass.Files, pass.Fset), nil
}

func ParseDirectives(files []*ast.File, fset *token.FileSet) []Directive {
	var dirs []Directive
	for _, f := range files {
		// OPT(dh): in our old code, we skip all the commentmap work if we
		// couldn't find any directives, benchmark if that's actually
		// worth doing
		cm := ast.NewCommentMap(fset, f, f.Comments)
		for node, cgs := range cm {
			for _, cg := range cgs {
				for _, c := range cg.List {
					if !strings.HasPrefix(c.Text, "//lint:") {
						continue
					}
					cmd, args := parseDirective(c.Text)
					d := Directive{
						Command:   cmd,
						Arguments: args,
						Directive: c,
						Node:      node,
					}
					dirs = append(dirs, d)
				}
			}
		}
	}
	return dirs
}

// duplicated from report.DisplayPosition to break import cycle
func displayPosition(fset *token.FileSet, p token.Pos) token.Position {
	if p == token.NoPos {
		return token.Position{}
	}

	// Only use the adjusted position if it points to another Go file.
	// This means we'll point to the original file for cgo files, but
	// we won't point to a YACC grammar file.
	pos := fset.PositionFor(p, false)
	adjPos := fset.PositionFor(p, true)

	if filepath.Ext(adjPos.Filename) == ".go" {
		return adjPos
	}

	return pos
}

var Directives = &analysis.Analyzer{
	Name:             "directives",
	Doc:              "extracts linter directives",
	Run:              directives,
	RunDespiteErrors: true,
	ResultType:       reflect.TypeOf([]Directive{}),
}

func SerializeDirective(dir Directive, fset *token.FileSet) SerializedDirective {
	return SerializedDirective{
		Command:           dir.Command,
		Arguments:         dir.Arguments,
		DirectivePosition: displayPosition(fset, dir.Directive.Pos()),
		NodePosition:      displayPosition(fset, dir.Node.Pos()),
	}
}
