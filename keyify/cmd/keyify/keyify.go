package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/loader"
)

func parseOctothorpDecimal(s string) int {
	if s != "" && s[0] == '#' {
		if s, err := strconv.ParseInt(s[1:], 10, 32); err == nil {
			return int(s)
		}
	}
	return -1
}

func parsePos(pos string) (filename string, startOffset, endOffset int, err error) {
	if pos == "" {
		err = fmt.Errorf("no source position specified")
		return
	}

	colon := strings.LastIndex(pos, ":")
	if colon < 0 {
		err = fmt.Errorf("bad position syntax %q", pos)
		return
	}
	filename, offset := pos[:colon], pos[colon+1:]
	startOffset = -1
	endOffset = -1
	if hyphen := strings.Index(offset, ","); hyphen < 0 {
		// e.g. "foo.go:#123"
		startOffset = parseOctothorpDecimal(offset)
		endOffset = startOffset
	} else {
		// e.g. "foo.go:#123,#456"
		startOffset = parseOctothorpDecimal(offset[:hyphen])
		endOffset = parseOctothorpDecimal(offset[hyphen+1:])
	}
	if startOffset < 0 || endOffset < 0 {
		err = fmt.Errorf("invalid offset %q in query position", offset)
		return
	}
	return
}

func fileOffsetToPos(file *token.File, startOffset, endOffset int) (start, end token.Pos, err error) {
	// Range check [start..end], inclusive of both end-points.

	if 0 <= startOffset && startOffset <= file.Size() {
		start = file.Pos(int(startOffset))
	} else {
		err = fmt.Errorf("start position is beyond end of file")
		return
	}

	if 0 <= endOffset && endOffset <= file.Size() {
		end = file.Pos(int(endOffset))
	} else {
		err = fmt.Errorf("end position is beyond end of file")
		return
	}

	return
}

var (
	fRecursive bool
	fOneLine   bool
	fJSON      bool
)

func init() {
	flag.BoolVar(&fRecursive, "r", false, "keyify struct initializers recursively")
	flag.BoolVar(&fOneLine, "o", false, "print new struct initializer on a single line")
	flag.BoolVar(&fJSON, "json", false, "print new struct initializer as JSON")
}

func usage() {
	fmt.Printf("Usage: %s [flags] <position>\n\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	pos := flag.Args()[0]
	name, start, _, err := parsePos(pos)
	if err != nil {
		log.Fatal(err)
	}
	eval, err := filepath.EvalSymlinks(name)
	if err != nil {
		log.Fatal(err)
	}
	name, err = filepath.Abs(eval)
	if err != nil {
		log.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	bpkg, err := buildutil.ContainingPackage(&build.Default, cwd, name)
	if err != nil {
		log.Fatal(err)
	}
	conf := &loader.Config{}
	conf.ImportWithTests(bpkg.ImportPath)
	lprog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}
	var tf *token.File
	var af *ast.File
	pkg := lprog.InitialPackages()[0]
	for _, ff := range pkg.Files {
		file := lprog.Fset.File(ff.Pos())
		if file.Name() == name {
			af = ff
			tf = file
			break
		}
	}
	tstart, tend, err := fileOffsetToPos(tf, start, start)
	if err != nil {
		log.Fatal(err)
	}
	path, _ := astutil.PathEnclosingInterval(af, tstart, tend)
	var complit *ast.CompositeLit
	for _, p := range path {
		if p, ok := p.(*ast.CompositeLit); ok {
			complit = p
			break
		}
	}
	if complit == nil {
		log.Fatal("no composite literal found near point")
	}
	if len(complit.Elts) == 0 {
		printComplit(complit, complit, lprog.Fset, lprog.Fset)
		return
	}
	if _, ok := complit.Elts[0].(*ast.KeyValueExpr); ok {
		printComplit(complit, complit, lprog.Fset, lprog.Fset)
		return
	}
	st, ok := pkg.TypeOf(complit.Type).Underlying().(*types.Struct)
	if !ok {
		log.Fatal("not a struct initialiser")
		return
	}

	newFset := token.NewFileSet()
	newFile := newFset.AddFile("", -1, st.NumFields()+2)
	newComplit := &ast.CompositeLit{
		Type:   complit.Type,
		Lbrace: 1,
		Rbrace: token.Pos(st.NumFields() + 2),
	}
	newFile.AddLine(1)
	newFile.AddLine(st.NumFields() + 2)
	for i := 0; i < st.NumFields(); i++ {
		newFile.AddLine(2 + i)
		field := st.Field(i)
		elt := &ast.KeyValueExpr{
			Key:   &ast.Ident{NamePos: token.Pos(2 + i), Name: field.Name()},
			Value: copyExpr(complit.Elts[i], token.Pos(2+i)),
		}
		newComplit.Elts = append(newComplit.Elts, elt)
	}
	printComplit(complit, newComplit, lprog.Fset, newFset)
}

func printComplit(oldlit, newlit *ast.CompositeLit, oldfset, newfset *token.FileSet) {
	buf := &bytes.Buffer{}
	cfg := printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
	_ = cfg.Fprint(buf, newfset, newlit)
	if fJSON {
		output := struct {
			Start       int    `json:"start"`
			End         int    `json:"end"`
			Replacement string `json:"replacement"`
		}{
			oldfset.Position(oldlit.Pos()).Offset,
			oldfset.Position(oldlit.End()).Offset,
			buf.String(),
		}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Println(buf.String())
	}
}

func copyExpr(expr ast.Expr, line token.Pos) ast.Expr {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		cp := *expr
		cp.ValuePos = 0
		return &cp
	case *ast.BinaryExpr:
		cp := *expr
		cp.X = copyExpr(cp.X, line)
		cp.OpPos = 0
		cp.Y = copyExpr(cp.Y, line)
		return &cp
	case *ast.CallExpr:
		cp := *expr
		cp.Fun = copyExpr(cp.Fun, line)
		cp.Lparen = 0
		for i, v := range cp.Args {
			cp.Args[i] = copyExpr(v, line)
		}
		cp.Ellipsis = line
		cp.Rparen = 0
		return &cp
	case *ast.CompositeLit:
		cp := *expr
		cp.Type = copyExpr(cp.Type, line)
		cp.Lbrace = 0
		for i, v := range cp.Elts {
			cp.Elts[i] = copyExpr(v, line)
		}
		cp.Rbrace = 0
		return &cp
	case *ast.Ident:
		cp := *expr
		cp.NamePos = 0
		return &cp
	case *ast.IndexExpr:
		cp := *expr
		cp.X = copyExpr(cp.X, line)
		cp.Lbrack = 0
		cp.Index = copyExpr(cp.Index, line)
		cp.Rbrack = 0
		return &cp
	case *ast.ParenExpr:
		cp := *expr
		cp.Lparen = 0
		cp.X = copyExpr(cp.X, line)
		cp.Rparen = 0
		return &cp
	case *ast.SelectorExpr:
		cp := *expr
		cp.X = copyExpr(cp.X, line)
		cp.Sel = copyExpr(cp.Sel, line).(*ast.Ident)
		return &cp
	case *ast.SliceExpr:
		cp := *expr
		cp.X = copyExpr(cp.X, line)
		cp.Lbrack = 0
		cp.Low = copyExpr(cp.Low, line)
		cp.High = copyExpr(cp.High, line)
		cp.Max = copyExpr(cp.Max, line)
		cp.Rbrack = 0
		return &cp
	case *ast.StarExpr:
		cp := *expr
		cp.Star = 0
		cp.X = copyExpr(cp.X, line)
		return &cp
	case *ast.TypeAssertExpr:
		cp := *expr
		cp.X = copyExpr(cp.X, line)
		cp.Lparen = 0
		cp.Type = copyExpr(cp.Type, line)
		cp.Rparen = 0
		return &cp
	case *ast.UnaryExpr:
		cp := *expr
		cp.OpPos = 0
		cp.X = copyExpr(cp.X, line)
		return &cp
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("shouldn't happen: unknown ast.Expr of type %T", expr))
	}
	return nil
}
