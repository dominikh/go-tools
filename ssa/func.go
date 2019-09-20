// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssa

// This file implements the Function and BasicBlock types.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"io"
	"os"
	"strings"
)

// addEdge adds a control-flow graph edge from from to to.
func addEdge(from, to *BasicBlock) {
	from.Succs = append(from.Succs, to)
	to.Preds = append(to.Preds, from)
}

// Control returns the last instruction in the block.
func (b *BasicBlock) Control() Instruction {
	if len(b.Instrs) == 0 {
		return nil
	}
	return b.Instrs[len(b.Instrs)-1]
}

// Parent returns the function that contains block b.
func (b *BasicBlock) Parent() *Function { return b.parent }

// String returns a human-readable label of this block.
// It is not guaranteed unique within the function.
//
func (b *BasicBlock) String() string {
	return fmt.Sprintf("%d", b.Index)
}

// emit appends an instruction to the current basic block.
// If the instruction defines a Value, it is returned.
//
func (b *BasicBlock) emit(i Instruction) Value {
	i.setBlock(b)
	b.Instrs = append(b.Instrs, i)
	v, _ := i.(Value)
	return v
}

// predIndex returns the i such that b.Preds[i] == c or panics if
// there is none.
func (b *BasicBlock) predIndex(c *BasicBlock) int {
	for i, pred := range b.Preds {
		if pred == c {
			return i
		}
	}
	panic(fmt.Sprintf("no edge %s -> %s", c, b))
}

// hasPhi returns true if b.Instrs contains φ-nodes.
func (b *BasicBlock) hasPhi() bool {
	_, ok := b.Instrs[0].(*Phi)
	return ok
}

func (b *BasicBlock) Phis() []Instruction {
	return b.phis()
}

// phis returns the prefix of b.Instrs containing all the block's φ-nodes.
func (b *BasicBlock) phis() []Instruction {
	for i, instr := range b.Instrs {
		if _, ok := instr.(*Phi); !ok {
			return b.Instrs[:i]
		}
	}
	return nil // unreachable in well-formed blocks
}

// replacePred replaces all occurrences of p in b's predecessor list with q.
// Ordinarily there should be at most one.
//
func (b *BasicBlock) replacePred(p, q *BasicBlock) {
	for i, pred := range b.Preds {
		if pred == p {
			b.Preds[i] = q
		}
	}
}

// replaceSucc replaces all occurrences of p in b's successor list with q.
// Ordinarily there should be at most one.
//
func (b *BasicBlock) replaceSucc(p, q *BasicBlock) {
	for i, succ := range b.Succs {
		if succ == p {
			b.Succs[i] = q
		}
	}
}

// removePred removes all occurrences of p in b's
// predecessor list and φ-nodes.
// Ordinarily there should be at most one.
//
func (b *BasicBlock) removePred(p *BasicBlock) {
	phis := b.phis()

	// We must preserve edge order for φ-nodes.
	j := 0
	for i, pred := range b.Preds {
		if pred != p {
			b.Preds[j] = b.Preds[i]
			// Strike out φ-edge too.
			for _, instr := range phis {
				phi := instr.(*Phi)
				phi.Edges[j] = phi.Edges[i]
			}
			j++
		}
	}
	// Nil out b.Preds[j:] and φ-edges[j:] to aid GC.
	for i := j; i < len(b.Preds); i++ {
		b.Preds[i] = nil
		for _, instr := range phis {
			instr.(*Phi).Edges[i] = nil
		}
	}
	b.Preds = b.Preds[:j]
	for _, instr := range phis {
		phi := instr.(*Phi)
		phi.Edges = phi.Edges[:j]
	}
}

// Destinations associated with unlabelled for/switch/select stmts.
// We push/pop one of these as we enter/leave each construct and for
// each BranchStmt we scan for the innermost target of the right type.
//
type targets struct {
	tail         *targets // rest of stack
	_break       *BasicBlock
	_continue    *BasicBlock
	_fallthrough *BasicBlock
}

// Destinations associated with a labelled block.
// We populate these as labels are encountered in forward gotos or
// labelled statements.
//
type lblock struct {
	_goto     *BasicBlock
	_break    *BasicBlock
	_continue *BasicBlock
}

// labelledBlock returns the branch target associated with the
// specified label, creating it if needed.
//
func (f *Function) labelledBlock(label *ast.Ident) *lblock {
	lb := f.lblocks[label.Obj]
	if lb == nil {
		lb = &lblock{_goto: f.newBasicBlock(label.Name)}
		if f.lblocks == nil {
			f.lblocks = make(map[*ast.Object]*lblock)
		}
		f.lblocks[label.Obj] = lb
	}
	return lb
}

// addParam adds a (non-escaping) parameter to f.Params of the
// specified name, type and source position.
//
func (f *Function) addParam(name string, typ types.Type, pos token.Pos) *Parameter {
	var b *BasicBlock
	if len(f.Blocks) > 0 {
		b = f.Blocks[0]
	}
	v := &Parameter{
		name: name,
	}
	v.setBlock(b)
	v.setType(typ)
	v.setPos(pos)
	f.Params = append(f.Params, v)
	if b != nil {
		// There may be no blocks if this function has no body. We
		// still create params, but aren't interested in the
		// instruction.
		f.Blocks[0].Instrs = append(f.Blocks[0].Instrs, v)
	}
	return v
}

func (f *Function) addParamObj(obj types.Object) *Parameter {
	name := obj.Name()
	if name == "" {
		name = fmt.Sprintf("arg%d", len(f.Params))
	}
	param := f.addParam(name, obj.Type(), obj.Pos())
	param.object = obj
	return param
}

// addSpilledParam declares a parameter that is pre-spilled to the
// stack; the function body will load/store the spilled location.
// Subsequent lifting will eliminate spills where possible.
//
func (f *Function) addSpilledParam(obj types.Object) {
	param := f.addParamObj(obj)
	spill := &Alloc{Comment: obj.Name()}
	spill.setType(types.NewPointer(obj.Type()))
	spill.setPos(obj.Pos())
	f.objects[obj] = spill
	f.Locals = append(f.Locals, spill)
	f.emit(spill)
	emitStore(f, spill, param, 0)
	// f.emit(&Store{Addr: spill, Val: param})
}

// startBody initializes the function prior to generating SSA code for its body.
// Precondition: f.Type() already set.
//
func (f *Function) startBody() {
	entry := f.newBasicBlock("entry")
	f.currentBlock = entry
	f.objects = make(map[types.Object]Value) // needed for some synthetics, e.g. init
}

func (f *Function) exitBlock() {
	old := f.currentBlock

	f.Exit = f.newBasicBlock("exit")
	f.currentBlock = f.Exit

	ret := f.results()
	results := make([]Value, len(ret))
	// Run function calls deferred in this
	// function when explicitly returning from it.
	f.emit(new(RunDefers))
	for i, r := range ret {
		results[i] = emitLoad(f, r)
	}

	f.emit(&Return{Results: results})
	f.currentBlock = old
}

// createSyntacticParams populates f.Params and generates code (spills
// and named result locals) for all the parameters declared in the
// syntax.  In addition it populates the f.objects mapping.
//
// Preconditions:
// f.startBody() was called.
// Postcondition:
// len(f.Params) == len(f.Signature.Params) + (f.Signature.Recv() ? 1 : 0)
//
func (f *Function) createSyntacticParams(recv *ast.FieldList, functype *ast.FuncType) {
	// Receiver (at most one inner iteration).
	if recv != nil {
		for _, field := range recv.List {
			for _, n := range field.Names {
				f.addSpilledParam(f.Pkg.info.Defs[n])
			}
			// Anonymous receiver?  No need to spill.
			if field.Names == nil {
				f.addParamObj(f.Signature.Recv())
			}
		}
	}

	// Parameters.
	if functype.Params != nil {
		n := len(f.Params) // 1 if has recv, 0 otherwise
		for _, field := range functype.Params.List {
			for _, n := range field.Names {
				f.addSpilledParam(f.Pkg.info.Defs[n])
			}
			// Anonymous parameter?  No need to spill.
			if field.Names == nil {
				f.addParamObj(f.Signature.Params().At(len(f.Params) - n))
			}
		}
	}

	// Named results.
	if functype.Results != nil {
		for _, field := range functype.Results.List {
			// Implicit "var" decl of locals for named results.
			for _, n := range field.Names {
				f.namedResults = append(f.namedResults, f.addLocalForIdent(n))
			}
		}

		if len(f.namedResults) == 0 {
			sig := f.Signature.Results()
			for i := 0; i < sig.Len(); i++ {
				v := f.addLocal(sig.At(i).Type(), sig.At(i).Pos())
				v.Comment = fmt.Sprintf("ret.%d", i)
				f.implicitResults = append(f.implicitResults, v)
			}
		}
	}
}

func numberNodes(f *Function) {
	var base ID
	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			base++
			instr.setID(base)
		}
	}
}

// buildReferrers populates the def/use information in all non-nil
// Value.Referrers slice.
// Precondition: all such slices are initially empty.
func buildReferrers(f *Function) {
	var rands []*Value
	for _, b := range f.Blocks {
		for _, instr := range b.Instrs {
			rands = instr.Operands(rands[:0]) // recycle storage
			for _, rand := range rands {
				if r := *rand; r != nil {
					if ref := r.Referrers(); ref != nil {
						*ref = append(*ref, instr)
					}
				}
			}
		}
	}
}

func (f *Function) emitConsts() {
	if len(f.Blocks) == 0 {
		f.consts = nil
		return
	}

	if len(f.consts) == 0 {
		return
	} else if len(f.consts) <= 32 {
		f.emitConstsFew()
	} else {
		f.emitConstsMany()
	}
}

func (f *Function) emitConstsFew() {
	dedup := make([]*Const, 0, 32)
	for _, c := range f.consts {
		if len(*c.Referrers()) == 0 {
			continue
		}
		found := false
		for _, d := range dedup {
			if c.typ == d.typ && c.Value == d.Value {
				replaceAll(c, d)
				found = true
				break
			}
		}
		if !found {
			dedup = append(dedup, c)
		}
	}

	instrs := make([]Instruction, len(f.Blocks[0].Instrs)+len(dedup))
	for i, c := range dedup {
		instrs[i] = c
		c.setBlock(f.Blocks[0])
	}
	copy(instrs[len(dedup):], f.Blocks[0].Instrs)
	f.Blocks[0].Instrs = instrs
	f.consts = nil
}

func (f *Function) emitConstsMany() {
	type constKey struct {
		typ   types.Type
		value constant.Value
	}

	m := make(map[constKey]Value, len(f.consts))
	areNil := 0
	for i, c := range f.consts {
		if len(*c.Referrers()) == 0 {
			f.consts[i] = nil
			areNil++
			continue
		}

		k := constKey{
			typ:   c.typ,
			value: c.Value,
		}
		if dup, ok := m[k]; !ok {
			m[k] = c
		} else {
			f.consts[i] = nil
			areNil++
			replaceAll(c, dup)
		}
	}

	instrs := make([]Instruction, len(f.Blocks[0].Instrs)+len(f.consts)-areNil)
	i := 0
	for _, c := range f.consts {
		if c != nil {
			instrs[i] = c
			c.setBlock(f.Blocks[0])
			i++
		}
	}
	copy(instrs[i:], f.Blocks[0].Instrs)
	f.Blocks[0].Instrs = instrs
	f.consts = nil
}

// finishBody() finalizes the function after SSA code generation of its body.
func (f *Function) finishBody() {
	f.objects = nil
	f.currentBlock = nil
	f.lblocks = nil

	// Don't pin the AST in memory (except in debug mode).
	if n := f.syntax; n != nil && !f.debugInfo() {
		f.syntax = extentNode{n.Pos(), n.End()}
	}

	// Remove from f.Locals any Allocs that escape to the heap.
	j := 0
	for _, l := range f.Locals {
		if !l.Heap {
			f.Locals[j] = l
			j++
		}
	}
	// Nil out f.Locals[j:] to aid GC.
	for i := j; i < len(f.Locals); i++ {
		f.Locals[i] = nil
	}
	f.Locals = f.Locals[:j]

	optimizeBlocks(f)

	buildReferrers(f)

	buildDomTree(f)

	if f.Prog.mode&NaiveForm == 0 {
		lift(f)
	}

	// emit constants after lifting, because lifting may produce new constants.
	f.emitConsts()

	f.namedResults = nil // (used by lifting)
	f.implicitResults = nil

	numberNodes(f)

	defer f.wr.Close()
	f.wr.WriteFunc("start", "start", f)

	phiElim(f)
	f.wr.WriteFunc("phiElim", "phiElim", f)

	if f.Prog.mode&PrintFunctions != 0 {
		printMu.Lock()
		f.WriteTo(os.Stdout)
		printMu.Unlock()
	}

	if f.Prog.mode&SanityCheckFunctions != 0 {
		mustSanityCheck(f, nil)
	}
}

func phiElim(f *Function) {
	for {
		changed := false
		for _, b := range f.Blocks {
			for _, instr := range b.Instrs {
				phi, ok := instr.(*Phi)
				if !ok {
					continue
				}
				if len(*phi.Referrers()) == 0 {
					continue
				}
				var v0 Value
				elim := true
				for _, e := range phi.Edges {
					if e == phi {
						continue
					}
					if v0 == nil {
						v0 = e
					}
					if v0 != e {
						if v0, ok := v0.(*Const); ok {
							if e, ok := e.(*Const); ok {
								if v0.typ == e.typ && v0.Value == e.Value {
									continue
								}
							}
						}
						elim = false
						break
					}
				}
				if elim {
					changed = true
					for _, ref := range *phi.Referrers() {
						for _, op := range ref.Operands(nil) {
							if *op == phi {
								*op = v0
								// Const don't currently track referrers
								if _, ok := v0.(*Const); !ok {
									*v0.Referrers() = append(*v0.Referrers(), ref)
								}
							}
						}
					}
					*phi.Referrers() = nil
				}
			}
		}
		if !changed {
			break
		}
	}
}

func (f *Function) RemoveNilBlocks() {
	f.removeNilBlocks()
}

// removeNilBlocks eliminates nils from f.Blocks and updates each
// BasicBlock.Index.  Use this after any pass that may delete blocks.
//
func (f *Function) removeNilBlocks() {
	j := 0
	for _, b := range f.Blocks {
		if b != nil {
			b.Index = j
			f.Blocks[j] = b
			j++
		}
	}
	// Nil out f.Blocks[j:] to aid GC.
	for i := j; i < len(f.Blocks); i++ {
		f.Blocks[i] = nil
	}
	f.Blocks = f.Blocks[:j]
}

// SetDebugMode sets the debug mode for package pkg.  If true, all its
// functions will include full debug info.  This greatly increases the
// size of the instruction stream, and causes Functions to depend upon
// the ASTs, potentially keeping them live in memory for longer.
//
func (pkg *Package) SetDebugMode(debug bool) {
	// TODO(adonovan): do we want ast.File granularity?
	pkg.debug = debug
}

// debugInfo reports whether debug info is wanted for this function.
func (f *Function) debugInfo() bool {
	return f.Pkg != nil && f.Pkg.debug
}

// addNamedLocal creates a local variable, adds it to function f and
// returns it.  Its name and type are taken from obj.  Subsequent
// calls to f.lookup(obj) will return the same local.
//
func (f *Function) addNamedLocal(obj types.Object) *Alloc {
	l := f.addLocal(obj.Type(), obj.Pos())
	l.Comment = obj.Name()
	f.objects[obj] = l
	return l
}

func (f *Function) addLocalForIdent(id *ast.Ident) *Alloc {
	return f.addNamedLocal(f.Pkg.info.Defs[id])
}

// addLocal creates an anonymous local variable of type typ, adds it
// to function f and returns it.  pos is the optional source location.
//
func (f *Function) addLocal(typ types.Type, pos token.Pos) *Alloc {
	v := &Alloc{}
	v.setType(types.NewPointer(typ))
	v.setPos(pos)
	f.Locals = append(f.Locals, v)
	f.emit(v)
	return v
}

// lookup returns the address of the named variable identified by obj
// that is local to function f or one of its enclosing functions.
// If escaping, the reference comes from a potentially escaping pointer
// expression and the referent must be heap-allocated.
//
func (f *Function) lookup(obj types.Object, escaping bool) Value {
	if v, ok := f.objects[obj]; ok {
		if alloc, ok := v.(*Alloc); ok && escaping {
			alloc.Heap = true
		}
		return v // function-local var (address)
	}

	// Definition must be in an enclosing function;
	// plumb it through intervening closures.
	if f.parent == nil {
		panic("no ssa.Value for " + obj.String())
	}
	outer := f.parent.lookup(obj, true) // escaping
	v := &FreeVar{
		name:   obj.Name(),
		typ:    outer.Type(),
		pos:    outer.Pos(),
		outer:  outer,
		parent: f,
	}
	f.objects[obj] = v
	f.FreeVars = append(f.FreeVars, v)
	return v
}

// emit emits the specified instruction to function f.
func (f *Function) emit(instr Instruction) Value {
	return f.currentBlock.emit(instr)
}

// RelString returns the full name of this function, qualified by
// package name, receiver type, etc.
//
// The specific formatting rules are not guaranteed and may change.
//
// Examples:
//      "math.IsNaN"                  // a package-level function
//      "(*bytes.Buffer).Bytes"       // a declared method or a wrapper
//      "(*bytes.Buffer).Bytes$thunk" // thunk (func wrapping method; receiver is param 0)
//      "(*bytes.Buffer).Bytes$bound" // bound (func wrapping method; receiver supplied by closure)
//      "main.main$1"                 // an anonymous function in main
//      "main.init#1"                 // a declared init function
//      "main.init"                   // the synthesized package initializer
//
// When these functions are referred to from within the same package
// (i.e. from == f.Pkg.Object), they are rendered without the package path.
// For example: "IsNaN", "(*Buffer).Bytes", etc.
//
// All non-synthetic functions have distinct package-qualified names.
// (But two methods may have the same name "(T).f" if one is a synthetic
// wrapper promoting a non-exported method "f" from another package; in
// that case, the strings are equal but the identifiers "f" are distinct.)
//
func (f *Function) RelString(from *types.Package) string {
	// Anonymous?
	if f.parent != nil {
		// An anonymous function's Name() looks like "parentName$1",
		// but its String() should include the type/package/etc.
		parent := f.parent.RelString(from)
		for i, anon := range f.parent.AnonFuncs {
			if anon == f {
				return fmt.Sprintf("%s$%d", parent, 1+i)
			}
		}

		return f.name // should never happen
	}

	// Method (declared or wrapper)?
	if recv := f.Signature.Recv(); recv != nil {
		return f.relMethod(from, recv.Type())
	}

	// Thunk?
	if f.method != nil {
		return f.relMethod(from, f.method.Recv())
	}

	// Bound?
	if len(f.FreeVars) == 1 && strings.HasSuffix(f.name, "$bound") {
		return f.relMethod(from, f.FreeVars[0].Type())
	}

	// Package-level function?
	// Prefix with package name for cross-package references only.
	if p := f.pkg(); p != nil && p != from {
		return fmt.Sprintf("%s.%s", p.Path(), f.name)
	}

	// Unknown.
	return f.name
}

func (f *Function) relMethod(from *types.Package, recv types.Type) string {
	return fmt.Sprintf("(%s).%s", relType(recv, from), f.name)
}

// writeSignature writes to buf the signature sig in declaration syntax.
func writeSignature(buf *bytes.Buffer, from *types.Package, name string, sig *types.Signature, params []*Parameter) {
	buf.WriteString("func ")
	if recv := sig.Recv(); recv != nil {
		buf.WriteString("(")
		if n := params[0].Name(); n != "" {
			buf.WriteString(n)
			buf.WriteString(" ")
		}
		types.WriteType(buf, params[0].Type(), types.RelativeTo(from))
		buf.WriteString(") ")
	}
	buf.WriteString(name)
	types.WriteSignature(buf, sig, types.RelativeTo(from))
}

func (f *Function) pkg() *types.Package {
	if f.Pkg != nil {
		return f.Pkg.Pkg
	}
	return nil
}

var _ io.WriterTo = (*Function)(nil) // *Function implements io.Writer

func (f *Function) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	WriteFunction(&buf, f)
	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

// WriteFunction writes to buf a human-readable "disassembly" of f.
func WriteFunction(buf *bytes.Buffer, f *Function) {
	fmt.Fprintf(buf, "# Name: %s\n", f.String())
	if f.Pkg != nil {
		fmt.Fprintf(buf, "# Package: %s\n", f.Pkg.Pkg.Path())
	}
	if syn := f.Synthetic; syn != "" {
		fmt.Fprintln(buf, "# Synthetic:", syn)
	}
	if pos := f.Pos(); pos.IsValid() {
		fmt.Fprintf(buf, "# Location: %s\n", f.Prog.Fset.Position(pos))
	}

	if f.parent != nil {
		fmt.Fprintf(buf, "# Parent: %s\n", f.parent.Name())
	}

	from := f.pkg()

	if f.FreeVars != nil {
		buf.WriteString("# Free variables:\n")
		for i, fv := range f.FreeVars {
			fmt.Fprintf(buf, "# % 3d:\t%s %s\n", i, fv.Name(), relType(fv.Type(), from))
		}
	}

	if len(f.Locals) > 0 {
		buf.WriteString("# Locals:\n")
		for i, l := range f.Locals {
			fmt.Fprintf(buf, "# % 3d:\t%s %s\n", i, l.Name(), relType(deref(l.Type()), from))
		}
	}
	writeSignature(buf, from, f.Name(), f.Signature, f.Params)
	buf.WriteString(":\n")

	if f.Blocks == nil {
		buf.WriteString("\t(external)\n")
	}

	// NB. column calculations are confused by non-ASCII
	// characters and assume 8-space tabs.
	const punchcard = 80 // for old time's sake.
	const tabwidth = 8
	for _, b := range f.Blocks {
		if b == nil {
			// Corrupt CFG.
			fmt.Fprintf(buf, ".nil:\n")
			continue
		}
		n, _ := fmt.Fprintf(buf, "%d:", b.Index)
		bmsg := fmt.Sprintf("%s P:%d S:%d", b.Comment, len(b.Preds), len(b.Succs))
		fmt.Fprintf(buf, "%*s%s\n", punchcard-1-n-len(bmsg), "", bmsg)

		if false { // CFG debugging
			fmt.Fprintf(buf, "\t# CFG: %s --> %s --> %s\n", b.Preds, b, b.Succs)
		}
		for _, instr := range b.Instrs {
			buf.WriteString("\t")
			switch v := instr.(type) {
			case Value:
				l := punchcard - tabwidth
				// Left-align the instruction.
				if name := v.Name(); name != "" {
					n, _ := fmt.Fprintf(buf, "%s = ", name)
					l -= n
				}
				n, _ := buf.WriteString(instr.String())
				l -= n
				// Right-align the type if there's space.
				// XXX don't print the type anymore once we've updated all instructions to show their own type
				if t := v.Type(); t != nil {
					buf.WriteByte(' ')
					ts := relType(t, from)
					l -= len(ts) + len("  ") // (spaces before and after type)
					if l > 0 {
						fmt.Fprintf(buf, "%*s", l, "")
					}
					buf.WriteString(ts)
				}
			case nil:
				// Be robust against bad transforms.
				buf.WriteString("<deleted>")
			default:
				buf.WriteString(instr.String())
			}
			buf.WriteString("\n")
		}
	}
	fmt.Fprintf(buf, "\n")
}

// newBasicBlock adds to f a new basic block and returns it.  It does
// not automatically become the current block for subsequent calls to emit.
// comment is an optional string for more readable debugging output.
//
func (f *Function) newBasicBlock(comment string) *BasicBlock {
	b := &BasicBlock{
		Index:   len(f.Blocks),
		Comment: comment,
		parent:  f,
	}
	b.Succs = b.succs2[:0]
	f.Blocks = append(f.Blocks, b)
	return b
}

// NewFunction returns a new synthetic Function instance belonging to
// prog, with its name and signature fields set as specified.
//
// The caller is responsible for initializing the remaining fields of
// the function object, e.g. Pkg, Params, Blocks.
//
// It is practically impossible for clients to construct well-formed
// SSA functions/packages/programs directly, so we assume this is the
// job of the Builder alone.  NewFunction exists to provide clients a
// little flexibility.  For example, analysis tools may wish to
// construct fake Functions for the root of the callgraph, a fake
// "reflect" package, etc.
//
// TODO(adonovan): think harder about the API here.
//
func (prog *Program) NewFunction(name string, sig *types.Signature, provenance string) *Function {
	return &Function{Prog: prog, name: name, Signature: sig, Synthetic: provenance}
}

type extentNode [2]token.Pos

func (n extentNode) Pos() token.Pos { return n[0] }
func (n extentNode) End() token.Pos { return n[1] }

// Syntax returns an ast.Node whose Pos/End methods provide the
// lexical extent of the function if it was defined by Go source code
// (f.Synthetic==""), or nil otherwise.
//
// If f was built with debug information (see Package.SetDebugRef),
// the result is the *ast.FuncDecl or *ast.FuncLit that declared the
// function.  Otherwise, it is an opaque Node providing only position
// information; this avoids pinning the AST in memory.
//
func (f *Function) Syntax() ast.Node { return f.syntax }

func (f *Function) initHTML(name string) {
	if name == "" {
		return
	}
	if rel := f.RelString(nil); rel == name {
		f.wr = NewHTMLWriter("ssa.html", rel, "")
	}
}
