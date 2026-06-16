// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ir

// This file defines the Const SSA value type.

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"strconv"

	"honnef.co/go/tools/internal/xtools-internal/typesinternal"
)

// soleTypeKind returns a BasicInfo for which constant.Value can
// represent all zero values for the types in the type set.
//
//	types.IsBoolean for false is a representative.
//	types.IsInteger for 0
//	types.IsString for ""
//	0 otherwise.
func soleTypeKind(typ types.Type) types.BasicInfo {
	// State records the set of possible zero values (false, 0, "").
	// Candidates (perhaps all) are eliminated during the type-set
	// iteration, which executes at least once.
	state := types.IsBoolean | types.IsInteger | types.IsString
	underIs(typ, func(ut types.Type) bool {
		var c types.BasicInfo
		if t, ok := ut.(*types.Basic); ok {
			c = t.Info()
		}
		if c&types.IsNumeric != 0 { // int/float/complex
			c = types.IsInteger
		}
		state = state & c
		return state != 0
	})
	return state
}

// NewConst returns a new constant of the specified value and type.
// val must be valid according to the specification of Const.Value.
func NewConst(val constant.Value, typ types.Type, source ast.Node) *Const {
	if val == nil {
		switch soleTypeKind(typ) {
		case types.IsBoolean:
			val = constant.MakeBool(false)
		case types.IsInteger:
			val = constant.MakeInt64(0)
		case types.IsString:
			val = constant.MakeString("")
		}
	}
	c := &Const{
		register: register{
			typ: typ,
		},
		Value: val,
	}
	c.setSource(source)
	return c
}

// intConst returns an 'int' constant that evaluates to i.
// (i is an int64 in case the host is narrower than the target.)
func intConst(i int64, source ast.Node) *Const {
	return NewConst(constant.MakeInt64(i), tInt, source)
}

// nilConst returns a nil constant of the specified type, which may
// be any reference type, including interfaces.
func nilConst(typ types.Type, source ast.Node) *Const {
	return NewConst(nil, typ, source)
}

// stringConst returns a 'string' constant that evaluates to s.
func stringConst(s string, source ast.Node) *Const {
	return NewConst(constant.MakeString(s), tString, source)
}

// zeroConst returns a new "zero" constant of the specified type.
func zeroConst(t types.Type, source ast.Node) Constant {
	return NewConst(nil, t, source)
}

func (c *Const) RelString(from *types.Package) string {
	var p string
	if c.Value == nil {
		p, _ = typesinternal.ZeroString(c.typ, types.RelativeTo(from))
		p = "const " + p
	} else if c.Value.Kind() == constant.String {
		v := constant.StringVal(c.Value)
		const max = 20
		// TODO(adonovan): don't cut a rune in half.
		if len(v) > max {
			v = v[:max-3] + "..." // abbreviate
		}
		p = strconv.Quote(v)
	} else {
		p = c.Value.String()
	}
	return p + ":" + relType(c.Type(), from)
}

func (c *Const) String() string {
	if c.block == nil {
		// Constants don't have a block till late in the compilation process. But we want to print consts during
		// debugging.
		return c.RelString(nil)
	}
	return c.RelString(c.Parent().pkg())
}

func (v *AggregateConst) RelString(pkg *types.Package) string {
	var b bytes.Buffer
	fmt.Fprint(&b, "const {")
	for i, vv := range v.Values {
		if i > 0 {
			fmt.Fprint(&b, ", ")
		}
		fmt.Fprint(&b, relName(vv, v))
	}
	fmt.Fprint(&b, "}")
	return b.String()
}

func (v *AggregateConst) String() string {
	if v.block == nil {
		return v.RelString(nil)
	}
	return v.RelString(v.Parent().pkg())
}

// IsNil returns true if this constant represents a typed or untyped nil value.
func (c *Const) IsNil() bool {
	return c.Value == nil
}

// Int64 returns the numeric value of this constant truncated to fit
// a signed 64-bit integer.
func (c *Const) Int64() int64 {
	switch x := constant.ToInt(c.Value); x.Kind() {
	case constant.Int:
		if i, ok := constant.Int64Val(x); ok {
			return i
		}
		return 0
	case constant.Float:
		f, _ := constant.Float64Val(x)
		return int64(f)
	}
	panic(fmt.Sprintf("unexpected constant value: %T", c.Value))
}

// Uint64 returns the numeric value of this constant truncated to fit
// an unsigned 64-bit integer.
func (c *Const) Uint64() uint64 {
	switch x := constant.ToInt(c.Value); x.Kind() {
	case constant.Int:
		if u, ok := constant.Uint64Val(x); ok {
			return u
		}
		return 0
	case constant.Float:
		f, _ := constant.Float64Val(x)
		return uint64(f)
	}
	panic(fmt.Sprintf("unexpected constant value: %T", c.Value))
}

// Float64 returns the numeric value of this constant truncated to fit
// a float64.
func (c *Const) Float64() float64 {
	f, _ := constant.Float64Val(c.Value)
	return f
}

// Complex128 returns the complex value of this constant truncated to
// fit a complex128.
func (c *Const) Complex128() complex128 {
	re, _ := constant.Float64Val(constant.Real(c.Value))
	im, _ := constant.Float64Val(constant.Imag(c.Value))
	return complex(re, im)
}

func (c *Const) equal(o Constant) bool {
	// TODO(dh): don't use == for types, this will miss identical pointer types, among others
	oc, ok := o.(*Const)
	if !ok {
		return false
	}
	return c.typ == oc.typ && c.Value == oc.Value && c.source == oc.source
}

func (c *AggregateConst) equal(o Constant) bool {
	oc, ok := o.(*AggregateConst)
	if !ok {
		return false
	}
	// TODO(dh): don't use == for types, this will miss identical pointer types, among others
	if c.typ != oc.typ {
		return false
	}
	if c.source != oc.source {
		return false
	}
	for i, v := range c.Values {
		if !v.(Constant).equal(oc.Values[i].(Constant)) {
			return false
		}
	}
	return true
}
