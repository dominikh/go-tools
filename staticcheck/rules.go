package staticcheck

import (
	"errors"
	"fmt"
	"go/constant"
	"go/types"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"honnef.co/go/ssa"
	"honnef.co/go/staticcheck/vrp"
)

type ArgumentRule interface {
	Index() int
	Validate(ssa.Value, *ssa.Function, *Checker) error
}

type CallRule struct {
	Arguments []ArgumentRule
}

type argumentRule struct {
	idx     int
	Message string
}

func (a argumentRule) Index() int {
	return a.idx
}

type ValidRegexp struct {
	argumentRule
}

func extractConsts(v ssa.Value) []*ssa.Const {
	switch v := v.(type) {
	case *ssa.Const:
		return []*ssa.Const{v}
	case *ssa.Phi:
		ops := v.Operands(nil)
		var out []*ssa.Const
		for _, op := range ops {
			out = append(out, extractConsts(*op)...)
		}
		return out
	default:
		return nil
	}
}

func (vr ValidRegexp) Validate(v ssa.Value, _ *ssa.Function, _ *Checker) error {
	for _, c := range extractConsts(v) {
		if c.Value == nil {
			continue
		}
		if c.Value.Kind() != constant.String {
			continue
		}
		s := constant.StringVal(c.Value)
		if _, err := regexp.Compile(s); err != nil {
			return err
		}
	}
	return nil
}

type ValidTimeLayout struct {
	argumentRule
}

func (vt ValidTimeLayout) Validate(v ssa.Value, _ *ssa.Function, _ *Checker) error {
	for _, c := range extractConsts(v) {
		if c.Value == nil {
			continue
		}
		if c.Value.Kind() != constant.String {
			continue
		}
		s := constant.StringVal(c.Value)
		s = strings.Replace(s, "_", " ", -1)
		s = strings.Replace(s, "Z", "-", -1)
		_, err := time.Parse(s, s)
		if err != nil {
			return err
		}
	}
	return nil
}

type ValidURL struct {
	argumentRule
}

func (vt ValidURL) Validate(v ssa.Value, _ *ssa.Function, _ *Checker) error {
	for _, c := range extractConsts(v) {
		if c.Value == nil {
			continue
		}
		if c.Value.Kind() != constant.String {
			continue
		}
		s := constant.StringVal(c.Value)
		_, err := url.Parse(s)
		if err != nil {
			if vt.Message != "" {
				return errors.New(vt.Message)
			}
			return fmt.Errorf("%q is not a valid URL: %s", s, err)
		}
	}
	return nil
}

type NotIntValue struct {
	argumentRule
	Not vrp.Z
}

func (ni NotIntValue) Validate(v ssa.Value, fn *ssa.Function, c *Checker) error {
	r, ok := c.funcDescs.Get(fn).Ranges.Get(v).(vrp.IntInterval)
	if !ok || !r.IsKnown() {
		return nil
	}
	if r.Lower != r.Upper {
		return nil
	}
	if r.Lower.Cmp(ni.Not) == 0 {
		if ni.Message != "" {
			return errors.New(ni.Message)
		}
		return fmt.Errorf("argument mustn't be of value %s", ni.Not)
	}
	return nil
}

type ValidUTF8 struct {
	argumentRule
}

func (vu ValidUTF8) Validate(v ssa.Value, _ *ssa.Function, _ *Checker) error {
	for _, c := range extractConsts(v) {
		if c.Value == nil {
			continue
		}
		if c.Value.Kind() != constant.String {
			continue
		}
		s := constant.StringVal(c.Value)
		if !utf8.ValidString(s) {
			if vu.Message != "" {
				return errors.New(vu.Message)
			}
			return fmt.Errorf("%q is not a valid UTF-8 encoded string", s)
		}
	}
	return nil
}

type BufferedChannel struct {
	argumentRule
}

func (bc BufferedChannel) Validate(v ssa.Value, fn *ssa.Function, c *Checker) error {
	r, ok := c.funcDescs.Get(fn).Ranges[v].(vrp.ChannelInterval)
	if !ok || !r.IsKnown() {
		return nil
	}
	if r.Size.Lower.Cmp(vrp.NewZ(0)) == 0 &&
		r.Size.Upper.Cmp(vrp.NewZ(0)) == 0 {
		if bc.Message != "" {
			return errors.New(bc.Message)
		}
		return errors.New("the channel should be buffered")
	}
	return nil
}

type Pointer struct {
	argumentRule
}

func (p Pointer) Validate(v ssa.Value, fn *ssa.Function, c *Checker) error {
	switch v.Type().Underlying().(type) {
	case *types.Pointer, *types.Interface:
		return nil
	}
	if p.Message != "" {
		return errors.New(p.Message)
	}
	return errors.New("argument is expected to be a pointer")
}

type NotConvertedInt struct {
	argumentRule
}

func (ci NotConvertedInt) Validate(v ssa.Value, fn *ssa.Function, c *Checker) error {
	conv, ok := v.(*ssa.Convert)
	if !ok {
		return nil
	}
	b, ok := conv.X.Type().Underlying().(*types.Basic)
	if !ok {
		return nil
	}
	if (b.Info() & types.IsInteger) == 0 {
		return nil
	}
	if ci.Message != "" {
		return errors.New(ci.Message)
	}
	return errors.New("argument should not be a converted integer")
}

type CanBinaryMarshal struct {
	argumentRule
}

func validEncodingBinaryType(typ types.Type) bool {
	typ = typ.Underlying()
	switch typ := typ.(type) {
	case *types.Basic:
		switch typ.Kind() {
		case types.Uint8, types.Uint16, types.Uint32, types.Uint64,
			types.Int8, types.Int16, types.Int32, types.Int64,
			types.Float32, types.Float64, types.Complex64, types.Complex128, types.Invalid:
			return true
		}
		return false
	case *types.Struct:
		n := typ.NumFields()
		for i := 0; i < n; i++ {
			if !validEncodingBinaryType(typ.Field(i).Type()) {
				return false
			}
		}
		return true
	case *types.Array:
		return validEncodingBinaryType(typ.Elem())
	case *types.Interface:
		// we can't determine if it's a valid type or not
		return true
	}
	return false
}

func (bm CanBinaryMarshal) Validate(v ssa.Value, fn *ssa.Function, c *Checker) error {
	typ := v.Type().Underlying()
	if ttyp, ok := typ.(*types.Pointer); ok {
		typ = ttyp.Elem().Underlying()
	}
	if ttyp, ok := typ.(interface {
		Elem() types.Type
	}); ok {
		if _, ok := ttyp.(*types.Pointer); !ok {
			typ = ttyp.Elem()
		}
	}

	if validEncodingBinaryType(typ) {
		return nil
	}
	if bm.Message != "" {
		return errors.New(bm.Message)
	}
	return fmt.Errorf("value of type %s cannot be used with binary.Write", v.Type())
}
