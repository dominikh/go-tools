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
		if !lint.IsGenerated(f) {
			out = append(out, f)
		}
	}
	return out
}

func (c *Checker) Funcs() map[string]lint.Func {
	return map[string]lint.Func{
		"ST1000": c.CheckPackageComment,
		"ST1001": c.CheckDotImports,
		"ST1002": nil, // XXX missing/malformed comments for exported identifiers
		"ST1003": nil, // XXX underscores in names
		"ST1004": nil, // XXX incorrect initialisms
		"ST1005": c.CheckErrorStrings,
		"ST1006": c.CheckReceiverNames,
		"ST1007": c.CheckIncDec,
		"ST1008": c.CheckErrorReturn,
		"ST1009": c.CheckUnexportedReturn,
		"ST1010": c.CheckContextFirstArg,
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
			if j.IsInTest(f) {
				continue
			}
			if f.Doc != nil && len(f.Doc.List) > 0 {
				hasDocs = true
				// XXX check that comment is well-formed
			}
		}

		if !hasDocs {
			for _, f := range pkg.Info.Files {
				if j.IsInTest(f) {
					continue
				}
				j.Errorf(f, "at least one file in a package should have a package comment")
			}
		}
	}
}

func (c *Checker) CheckDotImports(j *lint.Job) {
	for _, pkg := range j.Program.Packages {
		for _, f := range pkg.Info.Files {
			for _, imp := range f.Imports {
				if imp.Name != nil && imp.Name.Name == "." && !j.IsInTest(f) {
					j.Errorf(imp, "should not use dot imports")
				}
			}
		}
	}
}

func (c *Checker) CheckIncDec(j *lint.Job) {
	fn := func(node ast.Node) bool {
		assign, ok := node.(*ast.AssignStmt)
		if !ok || (assign.Tok != token.ADD_ASSIGN && assign.Tok != token.SUB_ASSIGN) {
			return true
		}
		if (len(assign.Lhs) != 1 || len(assign.Rhs) != 1) ||
			!lint.IsIntLiteral(assign.Rhs[0], "1") {
			return true
		}

		suffix := ""
		switch assign.Tok {
		case token.ADD_ASSIGN:
			suffix = "++"
		case token.SUB_ASSIGN:
			suffix = "--"
		}

		j.Errorf(assign, "should replace %s with %s%s", j.Render(assign), j.Render(assign.Lhs[0]), suffix)
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
		if !ast.IsExported(fn.Name()) || j.IsInMain(fn) || j.IsInTest(fn) {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Recv() != nil && !ast.IsExported(lint.Dereference(sig.Recv().Type()).(*types.Named).Obj().Name()) {
			continue
		}
		res := sig.Results()
		for i := 0; i < res.Len(); i++ {
			if named, ok := lint.DereferenceR(res.At(i).Type()).(*types.Named); ok &&
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
					if lint.Dereference(recv.Type()) != T.Type() {
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
		if j.IsInTest(fn) {
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
				if !lint.IsCallTo(call.Common(), "errors.New") && !lint.IsCallTo(call.Common(), "fmt.Errorf") {
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
					// It could still be a single-word function name, though.
					j.Errorf(call, "error strings should not be capitalized")
				}
			}
		}
	}
}
