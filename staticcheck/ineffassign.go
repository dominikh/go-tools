package staticcheck

// TODO(dh): consider implementing this on top of the naive SSA form

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	. "honnef.co/go/tools/lint/lintdsl"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/cfg"
)

type ineffKind uint8

const (
	ineffRead ineffKind = iota
	ineffWrite
	ineffEscape
)

type ineffInstr struct {
	kind ineffKind
	obj  types.Object
	node ast.Node
	id   int
}

func (instr ineffInstr) String() string {
	verb := ""
	switch instr.kind {
	case ineffRead:
		verb = "read"
	case ineffWrite:
		verb = "write"
	case ineffEscape:
		verb = "escape"
	}
	return fmt.Sprintf("%s %s", verb, instr.obj)
}

type ineffassign struct {
	pass         *analysis.Pass
	blocks       []*ineffBlock
	namedReturns []types.Object
	hasDefer     bool
	// set of idents on the lhs of a channel receive in a select statement
	selectAssignments map[*ast.Ident]struct{}

	idCnt int
}

type ineffBlock struct {
	pass *analysis.Pass
	b    *cfg.Block
	objs map[types.Object][]ineffInstr
}

func (b *ineffBlock) def(ident *ast.Ident, id int) {
	if ident.Name == "_" {
		return
	}
	obj := b.pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return
	}
	if obj.Parent() == obj.Pkg().Scope() {
		// Don't track package level variables
		return
	}
	b.objs[obj] = append(b.objs[obj], ineffInstr{
		kind: ineffWrite,
		obj:  obj,
		node: ident,
		id:   id,
	})
}

func (b *ineffBlock) markRead(ident *ast.Ident, id int) {
	if ident.Name == "_" {
		return
	}

	obj := b.pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return
	}
	if obj.Pkg() == nil || obj.Parent() == obj.Pkg().Scope() {
		// Don't track package level variables
		return
	}
	b.objs[obj] = append(b.objs[obj], ineffInstr{
		kind: ineffRead,
		obj:  obj,
		node: ident,
		id:   id,
	})
}

func (b *ineffBlock) closure(ident *ast.Ident, id int) {
	if ident.Name == "_" {
		return
	}
	obj := b.pass.TypesInfo.ObjectOf(ident)
	if obj == nil {
		return
	}
	if obj.Pkg() == nil || obj.Parent() == obj.Pkg().Scope() {
		// Don't track package level variables
		return
	}
	b.objs[obj] = append(b.objs[obj], ineffInstr{
		kind: ineffEscape,
		obj:  obj,
		node: ident,
		id:   id,
	})
}

func (b *ineffassign) id() int {
	b.idCnt++
	return b.idCnt
}

func unparen(expr ast.Node) ast.Node {
	if expr, ok := expr.(*ast.ParenExpr); ok {
		return unparen(expr.X)
	}
	return expr
}

func isZeroConst(c constant.Value) bool {
	switch c.Kind() {
	case constant.Unknown:
		return false
	case constant.Bool:
		//lint:ignore S1002 we care about the specific value, not its boolness
		return constant.BoolVal(c) == false
	case constant.String:
		return constant.StringVal(c) == ""
	case constant.Int:
		v, ok := constant.Uint64Val(c)
		return ok && v == 0
	case constant.Float:
		v, ok := constant.Float64Val(c)
		return ok && v == 0
	case constant.Complex:
		c1, ok1 := constant.Float64Val(constant.Real(c))
		c2, ok2 := constant.Float64Val(constant.Imag(c))
		return ok1 && ok2 && c1 == 0 && c2 == 0
	default:
		panic("unreachable")
	}
}

func isZeroLiteral(pass *analysis.Pass, e ast.Expr) bool {
	switch e := unparen(e).(type) {
	case *ast.CallExpr:
		if !pass.TypesInfo.Types[e.Fun].IsType() {
			break
		}
		return isZeroLiteral(pass, e.Args[0])
	case *ast.SelectorExpr:
		return isZeroLiteral(pass, e.Sel)
	case *ast.Ident:
		obj := pass.TypesInfo.ObjectOf(e)

		if obj == types.Universe.Lookup("false") || obj == types.Universe.Lookup("true") {
			// 'true' isn't really a zero literal, but whether a
			// variable is initialized with true or false is up to
			// personal taste, and not worth flagging.
			return true
		}
		if obj == types.Universe.Lookup("nil") {
			return true
		}
		if c, ok := obj.(*types.Const); ok {
			return isZeroConst(c.Val())
		}
		return false
	case *ast.BasicLit:
		c := constant.MakeFromLiteral(e.Value, e.Kind, 0)
		return isZeroConst(c)
	}
	return false
}

func (ineff *ineffassign) uses(cb *cfg.Block, obj types.Object, seen []bool) bool {
	if seen[cb.Index] {
		return false
	}
	seen[cb.Index] = true
	b := ineff.blocks[cb.Index]
	stack := b.objs[obj]
	if len(stack) > 0 {
		// If this block writes a new value, then there is no way that
		// any successors can read the previous value.
		return stack[0].kind == ineffRead || stack[0].kind == ineffEscape
	}
	for _, succ := range cb.Succs {
		if ineff.uses(succ, obj, seen) {
			return true
		}
	}
	return false
}

func (ineff *ineffassign) process(pass *analysis.Pass, node ast.Node) {
	ineff.pass = pass
	var g *cfg.CFG
	var funcType *ast.FuncType
	switch node := node.(type) {
	case *ast.FuncDecl:
		g = pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs).FuncDecl(node)
		funcType = node.Type
	case *ast.FuncLit:
		g = pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs).FuncLit(node)
		funcType = node.Type
	default:
		panic(fmt.Sprintf("unsupported type %T", node))
	}

	if g == nil {
		return
	}

	fmt.Println(g.Format(pass.Fset))

	ineff.blocks = make([]*ineffBlock, len(g.Blocks))

	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			for _, name := range field.Names {
				ineff.namedReturns = append(ineff.namedReturns, pass.TypesInfo.ObjectOf(name))
			}
		}
	}

	// find all assignments made in select statements
	ineff.selectAssignments = map[*ast.Ident]struct{}{}
	ast.Inspect(node, func(node ast.Node) bool {
		if comm, ok := node.(*ast.CommClause); ok {
			if assign, ok := comm.Comm.(*ast.AssignStmt); ok {
				if ident, ok := assign.Lhs[0].(*ast.Ident); ok {
					ineff.selectAssignments[ident] = struct{}{}
				}
			}
		}
		return true
	})

	for _, b := range g.Blocks {
		// OPT(dh): can we skip dead blocks?
		ineff.processBlock(b)
	}

	escapes := map[types.Object]int{}
	for _, b := range ineff.blocks {
		for obj, instrs := range b.objs {
			for _, instr := range instrs {
				if instr.kind == ineffEscape {
					escapes[obj] = instr.id
				}
			}
		}
	}
	if ineff.hasDefer {
		// the function has a defer, which means panics might get
		// recovered from, which makes the values in named return
		// arguments relevant at all times.
		for _, obj := range ineff.namedReturns {
			escapes[obj] = 0
		}
	}

	scope := pass.TypesInfo.Scopes[funcType]
	tracked := map[types.Object]bool{}
	report := func(obj types.Object, node ast.Node) {
		b, ok := tracked[obj]
		if !ok {
			// check that the variable was defined in the function and
			// isn't a free variable.
			//
			// we need to explicitly check for obj.Parent() == scope
			// because a function's scope's extents are too small,
			// only including the function body. Thus, even though the
			// scope contains parameters, the following evaluates to
			// false: obj.Parent().Contains(obj.Pos())
			b = obj.Parent() == scope || scope.Contains(obj.Pos())
			tracked[obj] = b
		}
		if b {
			ReportNodef(pass, node, "this value of %s is never used", node)
		}
	}

	for _, cb := range g.Blocks {
		b := ineff.blocks[cb.Index]
		vars := b.objs
	objLoop:
		for obj, stack := range vars {
		instrLoop:
			for i, instr := range stack {
				if instr.kind == ineffWrite {
					if id, ok := escapes[obj]; ok && id < instr.id {
						// the value escaped before this write
						continue objLoop
					}

					if i < len(stack)-1 {
						if stack[i+1].kind == ineffRead || stack[i+1].kind == ineffEscape {
							continue instrLoop
						}
						// we're overwriting the value in the
						// same block, no successor block can
						// read the old value.
						report(obj, instr.node)
						continue instrLoop
					}

					// we're neither reading nor overwriting
					// in this block, so check successors
					seen := make([]bool, len(g.Blocks))
					for _, succ := range cb.Succs {
						if ineff.uses(succ, obj, seen) {
							continue instrLoop
						}
					}
					report(obj, instr.node)
				}
			}
		}
	}
}

func (ineff *ineffassign) processBlock(cb *cfg.Block) {
	b := &ineffBlock{
		pass: ineff.pass,
		b:    cb,
		objs: map[types.Object][]ineffInstr{},
	}
	ineff.blocks[cb.Index] = b

	markReadIdents := func(n ast.Node) {
		ast.Inspect(n, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncLit:
				ast.Inspect(n, func(n ast.Node) bool {
					if ident, ok := n.(*ast.Ident); ok {
						b.closure(ident, ineff.id())
					}
					return true
				})
				return false
			case *ast.Ident:
				b.markRead(n, ineff.id())
			case *ast.UnaryExpr:
				if n.Op == token.AND {
					switch x := unparen(n.X).(type) {
					case *ast.Ident:
						b.closure(x, ineff.id())
					case *ast.IndexExpr:
						if ident, ok := unparen(x.X).(*ast.Ident); ok {
							b.closure(ident, ineff.id())
						}
					case *ast.SelectorExpr:
						if ident, ok := unparen(x.X).(*ast.Ident); ok {
							b.closure(ident, ineff.id())
						}
					}
				}
			case *ast.SelectorExpr:
				// if this is a method on a pointer receiver, and the
				// ident is not a pointer, then it has its address
				// taken and escapes

				if sig, ok := ineff.pass.TypesInfo.ObjectOf(n.Sel).Type().Underlying().(*types.Signature); ok {
					if ident, ok := unparen(n.X).(*ast.Ident); ok {
						// if the method receiver isn't a pointer,
						// then calling a method on a variable
						// doesn't casue the variable to escape
						recv := sig.Recv()
						if recv == nil {
							break
						}
						if _, ok := recv.Type().Underlying().(*types.Pointer); !ok {
							break
						}
						if _, ok := ineff.pass.TypesInfo.TypeOf(ident).Underlying().(*types.Pointer); ok {
							// if the variable is already a pointer, then
							// calling a method on it doesn't cause the
							// variable to escape
							break
						}
						b.closure(ident, ineff.id())
					}
				}
			case *ast.SliceExpr:
				if ident, ok := n.X.(*ast.Ident); ok {
					if _, ok := ineff.pass.TypesInfo.TypeOf(ident).Underlying().(*types.Array); ok {
						b.closure(ident, ineff.id())
					}
				}
			}
			return true
		})
	}

	start := 0
	if len(cb.Nodes) > 0 {
		if ident, ok := cb.Nodes[0].(*ast.Ident); ok {
			if _, ok := ineff.selectAssignments[ident]; ok {
				// This block is the body of a CommClause of the form x =
				// <-x. We didn't process the assignment when we saw it;
				// instead we need to inject it at the front of the body.
				b.def(ident, ineff.id())
				start = 1
			}
		}
	}

	for _, n := range cb.Nodes[start:] {
		if e, ok := n.(*ast.ExprStmt); ok {
			n = e.X
		}

		n = unparen(n)
		// record definitions and uses
		switch n := n.(type) {
		case *ast.ValueSpec:
			for _, v := range n.Values {
				markReadIdents(v)
			}
			// We track zero-initialized values because we may need to
			// mark them as escaping.
			for _, name := range n.Names {
				b.def(name, ineff.id())
			}
			if len(n.Values) == 0 {
				// mark zero-initialised values as used
				for _, name := range n.Names {
					markReadIdents(name)
				}
			}
		case *ast.AssignStmt:
			for _, rhs := range n.Rhs {
				markReadIdents(rhs)
			}
			if n.Tok >= token.ADD_ASSIGN && n.Tok <= token.AND_NOT_ASSIGN {
				// x += 1 uses x
				if len(n.Lhs) != 1 {
					panic("internal error: expected exactly one variable on lhs")
				}
				markReadIdents(n.Lhs[0])
			}

			var isSelect bool
			if ident, ok := n.Lhs[0].(*ast.Ident); ok {
				_, isSelect = ineff.selectAssignments[ident]
			}
			for i, lhs := range n.Lhs {
				if ident, ok := unparen(lhs).(*ast.Ident); ok {
					if !isSelect {
						// If these assignments are the CommClauses of a
						// select statement, then only the Rhs comes into
						// effect at this point. The actual assignment only
						// happens if the respective case gets entered.
						//
						// The go/cfg package cannot model the semantics of
						// select and produces the following kind of control
						// flow for select statements:
						//
						// 	.0: # entry
						// 		x byte
						// 		x = <-ch1
						// 		x = <-ch2
						// 		succs: 2 3
						//
						// 	.2: # select.body
						// 		x
						// 		println(x)
						// 		succs: 1
						//
						// If we interpreted the control flow as presented by
						// go/cfg, we'd claim that x = <- ch1 is an
						// ineffective assignment.
						//
						// To work around this, we omit the x = <-ch1
						// assignment in the original block and
						// instead only emit a use on ch1. Similarly,
						// in the select.body block, we omit the use
						// of x, and instead omit a definition of x.

						b.def(ident, ineff.id())
						if len(n.Lhs) == len(n.Rhs) {
							if v, ok := n.Rhs[i].(*ast.Ident); ok && ineff.pass.TypesInfo.ObjectOf(v) == types.Universe.Lookup("nil") {
								// assigning nil to a value is sometimes seen as a hint to the garbage collector.
								markReadIdents(ident)
							}
						}

						if len(n.Lhs) == len(n.Rhs) && n.Tok == token.DEFINE && isZeroLiteral(ineff.pass, n.Rhs[i]) {
							markReadIdents(ident)
						}
					}
				} else {
					// the lhs is not an ident, so it may be an array
					// access or field access, which read the nested
					// idents involved.
					markReadIdents(lhs)
				}
			}
		case *ast.IncDecStmt:
			// use the old value to create the new value
			markReadIdents(n.X)
			if ident, ok := unparen(n.X).(*ast.Ident); ok {
				b.def(ident, ineff.id())
			}
		}

		// record uses
		switch n := n.(type) {
		case *ast.AssignStmt:
			// handled elsewhere
		case *ast.ValueSpec:
			// handled elsewhere
		case *ast.BasicLit:
			// nothing to do
		case *ast.IncDecStmt:
			// handled elsewhere

		case *ast.BinaryExpr,
			*ast.CallExpr,
			*ast.GoStmt,
			*ast.IndexExpr,
			*ast.SelectorExpr,
			*ast.SendStmt,
			*ast.SliceExpr,
			*ast.StarExpr,
			*ast.TypeAssertExpr,
			*ast.UnaryExpr,
			*ast.Ident:
			markReadIdents(n)
		case *ast.DeferStmt:
			ineff.hasDefer = true
			markReadIdents(n)
		case *ast.ReturnStmt:
			if len(n.Results) == 0 {
				// use named return parameters
				for _, obj := range ineff.namedReturns {
					b.objs[obj] = append(b.objs[obj], ineffInstr{
						kind: ineffRead,
						obj:  obj,
						node: nil,
					})
				}
			} else {
				markReadIdents(n)
			}

		case *ast.CompositeLit:
			markReadIdents(n)
		case *ast.EmptyStmt:
			// nothing to do
		default:
			panic(fmt.Sprintf("internal error: unhandled AST node %T", n))
		}
	}
}
