package stylecheck // import "honnef.co/go/tools/stylecheck"

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"honnef.co/go/tools/lint"
	. "honnef.co/go/tools/lint/lintdsl"
	"honnef.co/go/tools/ssa"

	"golang.org/x/tools/go/types/typeutil"
)

type Checker struct {
	CheckGenerated bool
}

func NewChecker() *Checker {
	return &Checker{}
}

func (*Checker) Name() string   { return "stylecheck" }
func (*Checker) Prefix() string { return "ST" }

func (c *Checker) Init(prog *lint.Program) {
}

func (c *Checker) filterGenerated(files []*ast.File) []*ast.File {
	if c.CheckGenerated {
		return files
	}
	var out []*ast.File
	for _, f := range files {
		if !IsGenerated(f) {
			out = append(out, f)
		}
	}
	return out
}

func (c *Checker) Funcs() map[string]lint.Func {
	return map[string]lint.Func{
		"ST1000": c.CheckPackageComment,
		"ST1001": c.CheckDotImports,
		"ST1002": c.CheckBlankImports,
		"ST1003": c.CheckNames,
		"ST1004": nil, // XXX
		"ST1005": c.CheckErrorStrings,
		"ST1006": c.CheckReceiverNames,
		"ST1007": c.CheckIncDec,
		"ST1008": c.CheckErrorReturn,
		"ST1009": c.CheckUnexportedReturn,
		"ST1010": c.CheckContextFirstArg,
		"ST1011": c.CheckTimeNames,
		"ST1012": c.CheckErrorVarNames,
	}
}

func (c *Checker) CheckPackageComment(j *lint.Job) {
	// - At least one file in a package should have a package comment
	//
	// - For non-main packages, the comment should be of the form
	// "Package x ...". This has a slight potential for false
	// positives, as multiple files can have package comments, in
	// which case they get appended. But that doesn't happen a lot in
	// the real world.

	for _, pkg := range j.Program.Packages {
		hasDocs := false
		for _, f := range pkg.Info.Files {
			if IsInTest(j, f) {
				continue
			}
			if f.Doc != nil && len(f.Doc.List) > 0 {
				hasDocs = true
				if f.Name.Name != "main" {
					prefix := "Package " + f.Name.Name + " "
					if !strings.HasPrefix(strings.TrimSpace(f.Doc.Text()), prefix) {
						j.Errorf(f.Doc, `package comment should be of the form "%s..."`, prefix)
					}
				}
				f.Doc.Text()
			}
		}

		if !hasDocs {
			for _, f := range pkg.Info.Files {
				if IsInTest(j, f) {
					continue
				}
				j.Errorf(f, "at least one file in a package should have a package comment")
			}
		}
	}
}

func (c *Checker) CheckDotImports(j *lint.Job) {
	// TODO(dh): implement user-provided whitelist for dot imports
	for _, f := range c.filterGenerated(j.Program.Files) {
		for _, imp := range f.Imports {
			if imp.Name != nil && imp.Name.Name == "." && !IsInTest(j, f) {
				j.Errorf(imp, "should not use dot imports")
			}
		}
	}
}

func (c *Checker) CheckBlankImports(j *lint.Job) {
	fset := j.Program.Prog.Fset
	for _, f := range c.filterGenerated(j.Program.Files) {
		if IsInMain(j, f) || IsInTest(j, f) {
			continue
		}

		// Collect imports of the form `import _ "foo"`, i.e. with no
		// parentheses, as their comment will be associated with the
		// (paren-free) GenDecl, not the import spec itself.
		//
		// We don't directly process the GenDecl so that we can
		// correctly handle the following:
		//
		//  import _ "foo"
		//  import _ "bar"
		//
		// where only the first import should get flagged.
		skip := map[ast.Spec]bool{}
		ast.Inspect(f, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.File:
				return true
			case *ast.GenDecl:
				if node.Tok != token.IMPORT {
					return false
				}
				if node.Lparen == token.NoPos && node.Doc != nil {
					skip[node.Specs[0]] = true
				}
				return false
			}
			return false
		})
		for i, imp := range f.Imports {
			pos := fset.Position(imp.Pos())

			if !IsBlank(imp.Name) {
				continue
			}
			// Only flag the first blank import in a group of imports,
			// or don't flag any of them, if the first one is
			// commented
			if i > 0 {
				prev := f.Imports[i-1]
				prevPos := fset.Position(prev.Pos())
				if pos.Line-1 == prevPos.Line && IsBlank(prev.Name) {
					continue
				}
			}

			if imp.Doc == nil && imp.Comment == nil && !skip[imp] {
				j.Errorf(imp, "a blank import should be only in a main or test package, or have a comment justifying it")
			}
		}
	}
}

func (c *Checker) CheckIncDec(j *lint.Job) {
	// TODO(dh): this can be noisy for function bodies that look like this:
	// 	x += 3
	// 	...
	// 	x += 2
	// 	...
	// 	x += 1
	fn := func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || (assign.Tok != token.ADD_ASSIGN && assign.Tok != token.SUB_ASSIGN) {
			return true
		}
		if (len(assign.Lhs) != 1 || len(assign.Rhs) != 1) ||
			!IsIntLiteral(assign.Rhs[0], "1") {
			return true
		}

		suffix := ""
		switch assign.Tok {
		case token.ADD_ASSIGN:
			suffix = "++"
		case token.SUB_ASSIGN:
			suffix = "--"
		}

		j.Errorf(assign, "should replace %s with %s%s", Render(j, assign), Render(j, assign.Lhs[0]), suffix)
		return true
	}
	for _, f := range c.filterGenerated(j.Program.Files) {
		ast.Inspect(f, fn)
	}
}

func (c *Checker) CheckErrorReturn(j *lint.Job) {
fnLoop:
	for _, fn := range j.Program.InitialFunctions {
		sig := fn.Type().(*types.Signature)
		rets := sig.Results()
		if rets == nil || rets.Len() < 2 {
			continue
		}

		if rets.At(rets.Len()-1).Type() == types.Universe.Lookup("error").Type() {
			// Last return type is error. If the function also returns
			// errors in other positions, that's fine.
			continue
		}
		for i := rets.Len() - 2; i >= 0; i-- {
			if rets.At(i).Type() == types.Universe.Lookup("error").Type() {
				j.Errorf(rets.At(i), "error should be returned as the last argument")
				continue fnLoop
			}
		}
	}
}

// CheckUnexportedReturn checks that exported functions on exported
// types do not return unexported types.
func (c *Checker) CheckUnexportedReturn(j *lint.Job) {
	for _, fn := range j.Program.InitialFunctions {
		if fn.Synthetic != "" || fn.Parent() != nil {
			continue
		}
		if !ast.IsExported(fn.Name()) || IsInMain(j, fn) || IsInTest(j, fn) {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Recv() != nil && !ast.IsExported(Dereference(sig.Recv().Type()).(*types.Named).Obj().Name()) {
			continue
		}
		res := sig.Results()
		for i := 0; i < res.Len(); i++ {
			if named, ok := DereferenceR(res.At(i).Type()).(*types.Named); ok &&
				!ast.IsExported(named.Obj().Name()) &&
				named != types.Universe.Lookup("error").Type() {
				j.Errorf(fn, "should not return unexported type")
			}
		}
	}
}

func (c *Checker) CheckReceiverNames(j *lint.Job) {
	for _, pkg := range j.Program.Packages {
		for _, m := range pkg.Members {
			names := map[string]int{}

			var firstFn *types.Func
			if T, ok := m.Object().(*types.TypeName); ok {
				ms := typeutil.IntuitiveMethodSet(T.Type(), nil)
				for _, sel := range ms {
					fn := sel.Obj().(*types.Func)
					recv := fn.Type().(*types.Signature).Recv()
					if Dereference(recv.Type()) != T.Type() {
						// skip embedded methods
						continue
					}
					if firstFn == nil {
						firstFn = fn
					}
					if recv.Name() != "" && recv.Name() != "_" {
						names[recv.Name()]++
					}
					if recv.Name() == "self" || recv.Name() == "this" {
						j.Errorf(recv, `receiver name should be a reflection of its identity; don't use generic names such as "this" or "self"`)
					}
					if recv.Name() == "_" {
						j.Errorf(recv, "receiver name should not be an underscore, omit the name if it is unused")
					}
				}
			}

			if len(names) > 1 {
				var seen []string
				for name, count := range names {
					seen = append(seen, fmt.Sprintf("%dx %q", count, name))
				}

				j.Errorf(firstFn, "methods on the same type should have the same receiver name (seen %s)", strings.Join(seen, ", "))
			}
		}
	}
}

func (c *Checker) CheckContextFirstArg(j *lint.Job) {
	// TODO(dh): this check doesn't apply to test helpers. Example from the stdlib:
	// 	func helperCommandContext(t *testing.T, ctx context.Context, s ...string) (cmd *exec.Cmd) {
fnLoop:
	for _, fn := range j.Program.InitialFunctions {
		if fn.Synthetic != "" || fn.Parent() != nil {
			continue
		}
		params := fn.Signature.Params()
		if params.Len() < 2 {
			continue
		}
		if types.TypeString(params.At(0).Type(), nil) == "context.Context" {
			continue
		}
		for i := 1; i < params.Len(); i++ {
			param := params.At(i)
			if types.TypeString(param.Type(), nil) == "context.Context" {
				j.Errorf(param, "context.Context should be the first argument of a function")
				continue fnLoop
			}
		}
	}
}

func (c *Checker) CheckErrorStrings(j *lint.Job) {
	for _, fn := range j.Program.InitialFunctions {
		if IsInTest(j, fn) {
			// We don't care about malformed error messages in tests;
			// they're usually for direct human consumption, not part
			// of an API
			continue
		}
		for _, block := range fn.Blocks {
			for _, ins := range block.Instrs {
				call, ok := ins.(*ssa.Call)
				if !ok {
					continue
				}
				if !IsCallTo(call.Common(), "errors.New") && !IsCallTo(call.Common(), "fmt.Errorf") {
					continue
				}

				k, ok := call.Common().Args[0].(*ssa.Const)
				if !ok {
					continue
				}

				s := constant.StringVal(k.Value)
				if len(s) == 0 {
					continue
				}
				switch s[len(s)-1] {
				case '.', ':', '!', '\n':
					j.Errorf(call, "error strings should not end with punctuation or a newline")
				}
				idx := strings.IndexByte(s, ' ')
				if idx == -1 {
					// single word error message, probably not a real
					// error but something used in tests or during
					// debugging
					continue
				}
				word := s[:idx]
				first, _ := utf8.DecodeRuneInString(word)
				if !unicode.IsUpper(first) {
					continue
				}
				many := false
				for _, c := range []rune(word)[1:] {
					if unicode.IsUpper(c) {
						many = true
						break
					}
				}
				if !many {
					// First word in error starts with a capital
					// letter, and the word doesn't contain any other
					// capitals, making it unlikely to be an
					// initialism or multi-word function name.
					//
					// It could still be a single-word function name
					// or a proper noun, though.
					//
					// TODO(dh): example from the stdlib that we incorrectly flag:
					// 	func (w *pooledFlateWriter) Write(p []byte) (n int, err error) {
					// 		...
					// 		return 0, errors.New("Write after Close")
					// 		...
					// 	}

					j.Errorf(call, "error strings should not be capitalized")
				}
			}
		}
	}
}

func (c *Checker) CheckTimeNames(j *lint.Job) {
	suffixes := []string{
		"Sec", "Secs", "Seconds",
		"Msec", "Msecs",
		"Milli", "Millis", "Milliseconds",
		"Usec", "Usecs", "Microseconds",
		"MS", "Ms",
	}
	fn := func(T types.Type, names []*ast.Ident) {
		if !IsType(T, "time.Duration") && !IsType(T, "*time.Duration") {
			return
		}
		for _, name := range names {
			for _, suffix := range suffixes {
				if strings.HasSuffix(name.Name, suffix) {
					j.Errorf(name, "var %s is of type %v; don't use unit-specific suffix %q", name.Name, T, suffix)
					break
				}
			}
		}
	}
	for _, f := range j.Program.Files {
		ast.Inspect(f, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.ValueSpec:
				T := j.Program.Info.TypeOf(node.Type)
				fn(T, node.Names)
			case *ast.FieldList:
				for _, field := range node.List {
					T := j.Program.Info.TypeOf(field.Type)
					fn(T, field.Names)
				}
			}
			return true
		})
	}
}

func (c *Checker) CheckErrorVarNames(j *lint.Job) {
	for _, f := range j.Program.Files {
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				spec := spec.(*ast.ValueSpec)
				if len(spec.Names) != len(spec.Values) {
					continue
				}

				for i, name := range spec.Names {
					val := spec.Values[i]
					if !IsCallToAST(j, val, "errors.New") && !IsCallToAST(j, val, "fmt.Errorf") {
						continue
					}

					prefix := "err"
					if name.IsExported() {
						prefix = "Err"
					}
					if !strings.HasPrefix(name.Name, prefix) {
						j.Errorf(name, "error var %s should have name of the form %sFoo", name.Name, prefix)
					}
				}
			}
		}
	}
}
