package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/loader"
)

var (
	fRecursive bool
	fOneLine   bool
	fJSON      bool
	fMinify    bool
)

func init() {
	flag.BoolVar(&fRecursive, "r", false, "keyify struct initializers recursively")
	flag.BoolVar(&fOneLine, "o", false, "print new struct initializer on a single line")
	flag.BoolVar(&fJSON, "json", false, "print new struct initializer as JSON")
	flag.BoolVar(&fMinify, "m", false, "omit fields that are set to their zero value")
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
	conf.TypeCheckFuncBodies = func(s string) bool {
		return s == bpkg.ImportPath || s == bpkg.ImportPath+"_test"
	}
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
		lit := complit
		if fOneLine {
			lit = copyExpr(complit, 1).(*ast.CompositeLit)
		}
		printComplit(complit, lit, lprog.Fset, lprog.Fset)
		return
	}
	st, ok := pkg.TypeOf(complit.Type).Underlying().(*types.Struct)
	if !ok {
		log.Fatal("not a struct initialiser")
		return
	}

	var calcPos func(int) token.Pos
	if fOneLine {
		calcPos = func(int) token.Pos { return token.Pos(1) }
	} else {
		calcPos = func(i int) token.Pos { return token.Pos(2 + i) }
	}

	newFset := token.NewFileSet()
	newFile := newFset.AddFile("", -1, st.NumFields()+2)
	newComplit := &ast.CompositeLit{
		Type:   complit.Type,
		Lbrace: 1,
		Rbrace: token.Pos(st.NumFields() + 2),
	}
	if fOneLine {
		newComplit.Rbrace = 1
	}
	newFile.AddLine(1)
	newFile.AddLine(st.NumFields() + 2)
	n := 0
	for i := 0; i < st.NumFields(); i++ {
		newFile.AddLine(2 + n)
		field := st.Field(i)
		val := complit.Elts[i]
		if fMinify && isZero(val, pkg) {
			continue
		}
		elt := &ast.KeyValueExpr{
			Key:   &ast.Ident{NamePos: calcPos(n), Name: field.Name()},
			Value: copyExpr(val, calcPos(n)),
		}
		newComplit.Elts = append(newComplit.Elts, elt)
		n++
	}
	printComplit(complit, newComplit, lprog.Fset, newFset)
}

func isZero(val ast.Expr, pkg *loader.PackageInfo) bool {
	switch val := val.(type) {
	case *ast.BasicLit:
		switch val.Value {
		case `""`, "``", "0", "0.0", "0i", "0.":
			return true
		default:
			return false
		}
	case *ast.Ident:
		if _, ok := pkg.ObjectOf(val).(*types.Nil); ok {
			return true
		}
		if c, ok := pkg.ObjectOf(val).(*types.Const); ok {
			if c.Val().Kind() != constant.Bool {
				return false
			}
			return !constant.BoolVal(c.Val())
		}
		return false
	case *ast.CompositeLit:
		typ := pkg.TypeOf(val.Type)
		if typ == nil {
			return false
		}
		_, ok1 := typ.Underlying().(*types.Struct)
		_, ok2 := typ.Underlying().(*types.Array)
		return (ok1 || ok2) && len(val.Elts) == 0
	}
	return false
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
		if cp.Ellipsis != 0 {
			cp.Ellipsis = line
		}
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
	case *ast.KeyValueExpr:
		cp := *expr
		cp.Key = copyExpr(cp.Key, line)
		cp.Colon = 0
		cp.Value = copyExpr(cp.Value, line)
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
	case *ast.MapType:
		cp := *expr
		cp.Map = 0
		cp.Key = copyExpr(cp.Key, line)
		cp.Value = copyExpr(cp.Value, line)
		return &cp
	case *ast.ArrayType:
		cp := *expr
		cp.Lbrack = 0
		cp.Len = copyExpr(cp.Len, line)
		cp.Elt = copyExpr(cp.Elt, line)
		return &cp
	case *ast.Ellipsis:
		cp := *expr
		cp.Elt = copyExpr(cp.Elt, line)
		cp.Ellipsis = line
		return &cp
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("shouldn't happen: unknown ast.Expr of type %T", expr))
	}
	return nil
}
