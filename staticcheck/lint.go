// Package staticcheck contains a linter for Go source code.
package staticcheck // import "honnef.co/go/tools/staticcheck"

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	htmltemplate "html/template"
	"net/http"
	"reflect"
	"regexp"
	"regexp/syntax"
	"sort"
	"strconv"
	"strings"
	texttemplate "text/template"
	"unicode"

	. "honnef.co/go/tools/arg"
	"honnef.co/go/tools/code"
	"honnef.co/go/tools/deprecated"
	"honnef.co/go/tools/edit"
	"honnef.co/go/tools/facts"
	"honnef.co/go/tools/functions"
	"honnef.co/go/tools/internal/passes/buildssa"
	"honnef.co/go/tools/internal/sharedcheck"
	"honnef.co/go/tools/lint"
	. "honnef.co/go/tools/lint/lintdsl"
	"honnef.co/go/tools/pattern"
	"honnef.co/go/tools/printf"
	"honnef.co/go/tools/report"
	"honnef.co/go/tools/ssa"
	"honnef.co/go/tools/ssa/ssautil"
	"honnef.co/go/tools/staticcheck/vrp"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

func checkSortSlice(call *Call) {
	c := call.Instr.Common().StaticCallee()
	arg := call.Args[0]

	T := arg.Value.Value.Type().Underlying()
	switch T.(type) {
	case *types.Interface:
		// we don't know.
		// TODO(dh): if the value is a phi node we can look at its edges
		if k, ok := arg.Value.Value.(*ssa.Const); ok && k.Value == nil {
			// literal nil, e.g. sort.Sort(nil, ...)
			arg.Invalid(fmt.Sprintf("cannot call %s on nil literal", c))
		}
	case *types.Slice:
		// this is fine
	default:
		// this is not fine
		arg.Invalid(fmt.Sprintf("%s must only be called on slices, was called on %s", c, T))
	}
}

func validRegexp(call *Call) {
	arg := call.Args[0]
	err := ValidateRegexp(arg.Value)
	if err != nil {
		arg.Invalid(err.Error())
	}
}

type runeSlice []rune

func (rs runeSlice) Len() int               { return len(rs) }
func (rs runeSlice) Less(i int, j int) bool { return rs[i] < rs[j] }
func (rs runeSlice) Swap(i int, j int)      { rs[i], rs[j] = rs[j], rs[i] }

func utf8Cutset(call *Call) {
	arg := call.Args[1]
	if InvalidUTF8(arg.Value) {
		arg.Invalid(MsgInvalidUTF8)
	}
}

func uniqueCutset(call *Call) {
	arg := call.Args[1]
	if !UniqueStringCutset(arg.Value) {
		arg.Invalid(MsgNonUniqueCutset)
	}
}

func unmarshalPointer(name string, arg int) CallCheck {
	return func(call *Call) {
		if !Pointer(call.Args[arg].Value) {
			call.Args[arg].Invalid(fmt.Sprintf("%s expects to unmarshal into a pointer, but the provided value is not a pointer", name))
		}
	}
}

func pointlessIntMath(call *Call) {
	if ConvertedFromInt(call.Args[0].Value) {
		call.Invalid(fmt.Sprintf("calling %s on a converted integer is pointless", code.CallName(call.Instr.Common())))
	}
}

func checkValidHostPort(arg int) CallCheck {
	return func(call *Call) {
		if !ValidHostPort(call.Args[arg].Value) {
			call.Args[arg].Invalid(MsgInvalidHostPort)
		}
	}
}

var (
	checkRegexpRules = map[string]CallCheck{
		"regexp.MustCompile": validRegexp,
		"regexp.Compile":     validRegexp,
		"regexp.Match":       validRegexp,
		"regexp.MatchReader": validRegexp,
		"regexp.MatchString": validRegexp,
	}

	checkTimeParseRules = map[string]CallCheck{
		"time.Parse": func(call *Call) {
			arg := call.Args[Arg("time.Parse.layout")]
			err := ValidateTimeLayout(arg.Value)
			if err != nil {
				arg.Invalid(err.Error())
			}
		},
	}

	checkEncodingBinaryRules = map[string]CallCheck{
		"encoding/binary.Write": func(call *Call) {
			arg := call.Args[Arg("encoding/binary.Write.data")]
			if !CanBinaryMarshal(call.Pass, arg.Value) {
				arg.Invalid(fmt.Sprintf("value of type %s cannot be used with binary.Write", arg.Value.Value.Type()))
			}
		},
	}

	checkURLsRules = map[string]CallCheck{
		"net/url.Parse": func(call *Call) {
			arg := call.Args[Arg("net/url.Parse.rawurl")]
			err := ValidateURL(arg.Value)
			if err != nil {
				arg.Invalid(err.Error())
			}
		},
	}

	checkSyncPoolValueRules = map[string]CallCheck{
		"(*sync.Pool).Put": func(call *Call) {
			arg := call.Args[Arg("(*sync.Pool).Put.x")]
			typ := arg.Value.Value.Type()
			if !code.IsPointerLike(typ) {
				arg.Invalid("argument should be pointer-like to avoid allocations")
			}
		},
	}

	checkRegexpFindAllRules = map[string]CallCheck{
		"(*regexp.Regexp).FindAll":                    RepeatZeroTimes("a FindAll method", 1),
		"(*regexp.Regexp).FindAllIndex":               RepeatZeroTimes("a FindAll method", 1),
		"(*regexp.Regexp).FindAllString":              RepeatZeroTimes("a FindAll method", 1),
		"(*regexp.Regexp).FindAllStringIndex":         RepeatZeroTimes("a FindAll method", 1),
		"(*regexp.Regexp).FindAllStringSubmatch":      RepeatZeroTimes("a FindAll method", 1),
		"(*regexp.Regexp).FindAllStringSubmatchIndex": RepeatZeroTimes("a FindAll method", 1),
		"(*regexp.Regexp).FindAllSubmatch":            RepeatZeroTimes("a FindAll method", 1),
		"(*regexp.Regexp).FindAllSubmatchIndex":       RepeatZeroTimes("a FindAll method", 1),
	}

	checkUTF8CutsetRules = map[string]CallCheck{
		"strings.IndexAny":     utf8Cutset,
		"strings.LastIndexAny": utf8Cutset,
		"strings.ContainsAny":  utf8Cutset,
		"strings.Trim":         utf8Cutset,
		"strings.TrimLeft":     utf8Cutset,
		"strings.TrimRight":    utf8Cutset,
	}

	checkUniqueCutsetRules = map[string]CallCheck{
		"strings.Trim":      uniqueCutset,
		"strings.TrimLeft":  uniqueCutset,
		"strings.TrimRight": uniqueCutset,
	}

	checkUnmarshalPointerRules = map[string]CallCheck{
		"encoding/xml.Unmarshal":                unmarshalPointer("xml.Unmarshal", 1),
		"(*encoding/xml.Decoder).Decode":        unmarshalPointer("Decode", 0),
		"(*encoding/xml.Decoder).DecodeElement": unmarshalPointer("DecodeElement", 0),
		"encoding/json.Unmarshal":               unmarshalPointer("json.Unmarshal", 1),
		"(*encoding/json.Decoder).Decode":       unmarshalPointer("Decode", 0),
	}

	checkUnbufferedSignalChanRules = map[string]CallCheck{
		"os/signal.Notify": func(call *Call) {
			arg := call.Args[Arg("os/signal.Notify.c")]
			if UnbufferedChannel(arg.Value) {
				arg.Invalid("the channel used with signal.Notify should be buffered")
			}
		},
	}

	checkMathIntRules = map[string]CallCheck{
		"math.Ceil":  pointlessIntMath,
		"math.Floor": pointlessIntMath,
		"math.IsNaN": pointlessIntMath,
		"math.Trunc": pointlessIntMath,
		"math.IsInf": pointlessIntMath,
	}

	checkStringsReplaceZeroRules = map[string]CallCheck{
		"strings.Replace": RepeatZeroTimes("strings.Replace", 3),
		"bytes.Replace":   RepeatZeroTimes("bytes.Replace", 3),
	}

	checkListenAddressRules = map[string]CallCheck{
		"net/http.ListenAndServe":    checkValidHostPort(0),
		"net/http.ListenAndServeTLS": checkValidHostPort(0),
	}

	checkBytesEqualIPRules = map[string]CallCheck{
		"bytes.Equal": func(call *Call) {
			if ConvertedFrom(call.Args[Arg("bytes.Equal.a")].Value, "net.IP") &&
				ConvertedFrom(call.Args[Arg("bytes.Equal.b")].Value, "net.IP") {
				call.Invalid("use net.IP.Equal to compare net.IPs, not bytes.Equal")
			}
		},
	}

	checkRegexpMatchLoopRules = map[string]CallCheck{
		"regexp.Match":       loopedRegexp("regexp.Match"),
		"regexp.MatchReader": loopedRegexp("regexp.MatchReader"),
		"regexp.MatchString": loopedRegexp("regexp.MatchString"),
	}

	checkNoopMarshal = map[string]CallCheck{
		// TODO(dh): should we really flag XML? Even an empty struct
		// produces a non-zero amount of data, namely its type name.
		// Let's see if we encounter any false positives.
		//
		// Also, should we flag gob?
		"encoding/json.Marshal":           checkNoopMarshalImpl(Arg("json.Marshal.v"), "MarshalJSON", "MarshalText"),
		"encoding/xml.Marshal":            checkNoopMarshalImpl(Arg("xml.Marshal.v"), "MarshalXML", "MarshalText"),
		"(*encoding/json.Encoder).Encode": checkNoopMarshalImpl(Arg("(*encoding/json.Encoder).Encode.v"), "MarshalJSON", "MarshalText"),
		"(*encoding/xml.Encoder).Encode":  checkNoopMarshalImpl(Arg("(*encoding/xml.Encoder).Encode.v"), "MarshalXML", "MarshalText"),

		"encoding/json.Unmarshal":         checkNoopMarshalImpl(Arg("json.Unmarshal.v"), "UnmarshalJSON", "UnmarshalText"),
		"encoding/xml.Unmarshal":          checkNoopMarshalImpl(Arg("xml.Unmarshal.v"), "UnmarshalXML", "UnmarshalText"),
		"(*encoding/json.Decoder).Decode": checkNoopMarshalImpl(Arg("(*encoding/json.Decoder).Decode.v"), "UnmarshalJSON", "UnmarshalText"),
		"(*encoding/xml.Decoder).Decode":  checkNoopMarshalImpl(Arg("(*encoding/xml.Decoder).Decode.v"), "UnmarshalXML", "UnmarshalText"),
	}

	checkUnsupportedMarshal = map[string]CallCheck{
		"encoding/json.Marshal":           checkUnsupportedMarshalImpl(Arg("json.Marshal.v"), "json", "MarshalJSON", "MarshalText"),
		"encoding/xml.Marshal":            checkUnsupportedMarshalImpl(Arg("xml.Marshal.v"), "xml", "MarshalXML", "MarshalText"),
		"(*encoding/json.Encoder).Encode": checkUnsupportedMarshalImpl(Arg("(*encoding/json.Encoder).Encode.v"), "json", "MarshalJSON", "MarshalText"),
		"(*encoding/xml.Encoder).Encode":  checkUnsupportedMarshalImpl(Arg("(*encoding/xml.Encoder).Encode.v"), "xml", "MarshalXML", "MarshalText"),
	}

	checkAtomicAlignment = map[string]CallCheck{
		"sync/atomic.AddInt64":             checkAtomicAlignmentImpl,
		"sync/atomic.AddUint64":            checkAtomicAlignmentImpl,
		"sync/atomic.CompareAndSwapInt64":  checkAtomicAlignmentImpl,
		"sync/atomic.CompareAndSwapUint64": checkAtomicAlignmentImpl,
		"sync/atomic.LoadInt64":            checkAtomicAlignmentImpl,
		"sync/atomic.LoadUint64":           checkAtomicAlignmentImpl,
		"sync/atomic.StoreInt64":           checkAtomicAlignmentImpl,
		"sync/atomic.StoreUint64":          checkAtomicAlignmentImpl,
		"sync/atomic.SwapInt64":            checkAtomicAlignmentImpl,
		"sync/atomic.SwapUint64":           checkAtomicAlignmentImpl,
	}

	// TODO(dh): detect printf wrappers
	checkPrintfRules = map[string]CallCheck{
		"fmt.Errorf":  func(call *Call) { checkPrintfCall(call, 0, 1) },
		"fmt.Printf":  func(call *Call) { checkPrintfCall(call, 0, 1) },
		"fmt.Sprintf": func(call *Call) { checkPrintfCall(call, 0, 1) },
		"fmt.Fprintf": func(call *Call) { checkPrintfCall(call, 1, 2) },
	}

	checkSortSliceRules = map[string]CallCheck{
		"sort.Slice":         checkSortSlice,
		"sort.SliceIsSorted": checkSortSlice,
		"sort.SliceStable":   checkSortSlice,
	}

	checkWithValueKeyRules = map[string]CallCheck{
		"context.WithValue": checkWithValueKey,
	}
)

func checkPrintfCall(call *Call, fIdx, vIdx int) {
	f := call.Args[fIdx]
	var args []ssa.Value
	switch v := call.Args[vIdx].Value.Value.(type) {
	case *ssa.Slice:
		var ok bool
		args, ok = ssautil.Vararg(v)
		if !ok {
			// We don't know what the actual arguments to the function are
			return
		}
	case *ssa.Const:
		// nil, i.e. no arguments
	default:
		// We don't know what the actual arguments to the function are
		return
	}
	checkPrintfCallImpl(f, f.Value.Value, args)
}

type verbFlag int

const (
	isInt verbFlag = 1 << iota
	isBool
	isFP
	isString
	isPointer
	isPseudoPointer
	isSlice
	isAny
	noRecurse
)

var verbs = [...]verbFlag{
	'b': isPseudoPointer | isInt | isFP,
	'c': isInt,
	'd': isPseudoPointer | isInt,
	'e': isFP,
	'E': isFP,
	'f': isFP,
	'F': isFP,
	'g': isFP,
	'G': isFP,
	'o': isPseudoPointer | isInt,
	'p': isSlice | isPointer | noRecurse,
	'q': isInt | isString,
	's': isString,
	't': isBool,
	'T': isAny,
	'U': isInt,
	'v': isAny,
	'X': isPseudoPointer | isInt | isString,
	'x': isPseudoPointer | isInt | isString,
}

func checkPrintfCallImpl(carg *Argument, f ssa.Value, args []ssa.Value) {
	var msCache *typeutil.MethodSetCache
	if f.Parent() != nil {
		msCache = &f.Parent().Prog.MethodSets
	}

	elem := func(T types.Type, verb rune) ([]types.Type, bool) {
		if verbs[verb]&noRecurse != 0 {
			return []types.Type{T}, false
		}
		switch T := T.(type) {
		case *types.Slice:
			if verbs[verb]&isSlice != 0 {
				return []types.Type{T}, false
			}
			if verbs[verb]&isString != 0 && code.IsType(T.Elem().Underlying(), "byte") {
				return []types.Type{T}, false
			}
			return []types.Type{T.Elem()}, true
		case *types.Map:
			key := T.Key()
			val := T.Elem()
			return []types.Type{key, val}, true
		case *types.Struct:
			out := make([]types.Type, 0, T.NumFields())
			for i := 0; i < T.NumFields(); i++ {
				out = append(out, T.Field(i).Type())
			}
			return out, true
		case *types.Array:
			return []types.Type{T.Elem()}, true
		default:
			return []types.Type{T}, false
		}
	}
	isInfo := func(T types.Type, info types.BasicInfo) bool {
		basic, ok := T.Underlying().(*types.Basic)
		return ok && basic.Info()&info != 0
	}

	isStringer := func(T types.Type, ms *types.MethodSet) bool {
		sel := ms.Lookup(nil, "String")
		if sel == nil {
			return false
		}
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			// should be unreachable
			return false
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 0 {
			return false
		}
		if sig.Results().Len() != 1 {
			return false
		}
		if !code.IsType(sig.Results().At(0).Type(), "string") {
			return false
		}
		return true
	}
	isError := func(T types.Type, ms *types.MethodSet) bool {
		sel := ms.Lookup(nil, "Error")
		if sel == nil {
			return false
		}
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			// should be unreachable
			return false
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 0 {
			return false
		}
		if sig.Results().Len() != 1 {
			return false
		}
		if !code.IsType(sig.Results().At(0).Type(), "string") {
			return false
		}
		return true
	}

	isFormatter := func(T types.Type, ms *types.MethodSet) bool {
		sel := ms.Lookup(nil, "Format")
		if sel == nil {
			return false
		}
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			// should be unreachable
			return false
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 2 {
			return false
		}
		// TODO(dh): check the types of the arguments for more
		// precision
		if sig.Results().Len() != 0 {
			return false
		}
		return true
	}

	seen := map[types.Type]bool{}
	var checkType func(verb rune, T types.Type, top bool) bool
	checkType = func(verb rune, T types.Type, top bool) bool {
		if top {
			for k := range seen {
				delete(seen, k)
			}
		}
		if seen[T] {
			return true
		}
		seen[T] = true
		if int(verb) >= len(verbs) {
			// Unknown verb
			return true
		}

		flags := verbs[verb]
		if flags == 0 {
			// Unknown verb
			return true
		}

		ms := msCache.MethodSet(T)
		if isFormatter(T, ms) {
			// the value is responsible for formatting itself
			return true
		}

		if flags&isString != 0 && (isStringer(T, ms) || isError(T, ms)) {
			// Check for stringer early because we're about to dereference
			return true
		}

		T = T.Underlying()
		if flags&(isPointer|isPseudoPointer) == 0 && top {
			T = code.Dereference(T)
		}
		if flags&isPseudoPointer != 0 && top {
			t := code.Dereference(T)
			if _, ok := t.Underlying().(*types.Struct); ok {
				T = t
			}
		}

		if _, ok := T.(*types.Interface); ok {
			// We don't know what's in the interface
			return true
		}

		var info types.BasicInfo
		if flags&isInt != 0 {
			info |= types.IsInteger
		}
		if flags&isBool != 0 {
			info |= types.IsBoolean
		}
		if flags&isFP != 0 {
			info |= types.IsFloat | types.IsComplex
		}
		if flags&isString != 0 {
			info |= types.IsString
		}

		if info != 0 && isInfo(T, info) {
			return true
		}

		if flags&isString != 0 && (code.IsType(T, "[]byte") || isStringer(T, ms) || isError(T, ms)) {
			return true
		}

		if flags&isPointer != 0 && code.IsPointerLike(T) {
			return true
		}
		if flags&isPseudoPointer != 0 {
			switch U := T.Underlying().(type) {
			case *types.Pointer:
				if !top {
					return true
				}

				if _, ok := U.Elem().Underlying().(*types.Struct); !ok {
					return true
				}
			case *types.Chan, *types.Signature:
				return true
			}
		}

		if flags&isSlice != 0 {
			if _, ok := T.(*types.Slice); ok {
				return true
			}
		}

		if flags&isAny != 0 {
			return true
		}

		elems, ok := elem(T.Underlying(), verb)
		if !ok {
			return false
		}
		for _, elem := range elems {
			if !checkType(verb, elem, false) {
				return false
			}
		}

		return true
	}

	k, ok := f.(*ssa.Const)
	if !ok {
		return
	}
	actions, err := printf.Parse(constant.StringVal(k.Value))
	if err != nil {
		carg.Invalid("couldn't parse format string")
		return
	}

	ptr := 1
	hasExplicit := false

	checkStar := func(verb printf.Verb, star printf.Argument) bool {
		if star, ok := star.(printf.Star); ok {
			idx := 0
			if star.Index == -1 {
				idx = ptr
				ptr++
			} else {
				hasExplicit = true
				idx = star.Index
				ptr = star.Index + 1
			}
			if idx == 0 {
				carg.Invalid(fmt.Sprintf("Printf format %s reads invalid arg 0; indices are 1-based", verb.Raw))
				return false
			}
			if idx > len(args) {
				carg.Invalid(
					fmt.Sprintf("Printf format %s reads arg #%d, but call has only %d args",
						verb.Raw, idx, len(args)))
				return false
			}
			if arg, ok := args[idx-1].(*ssa.MakeInterface); ok {
				if !isInfo(arg.X.Type(), types.IsInteger) {
					carg.Invalid(fmt.Sprintf("Printf format %s reads non-int arg #%d as argument of *", verb.Raw, idx))
				}
			}
		}
		return true
	}

	// We only report one problem per format string. Making a
	// mistake with an index tends to invalidate all future
	// implicit indices.
	for _, action := range actions {
		verb, ok := action.(printf.Verb)
		if !ok {
			continue
		}

		if !checkStar(verb, verb.Width) || !checkStar(verb, verb.Precision) {
			return
		}

		off := ptr
		if verb.Value != -1 {
			hasExplicit = true
			off = verb.Value
		}
		if off > len(args) {
			carg.Invalid(
				fmt.Sprintf("Printf format %s reads arg #%d, but call has only %d args",
					verb.Raw, off, len(args)))
			return
		} else if verb.Value == 0 && verb.Letter != '%' {
			carg.Invalid(fmt.Sprintf("Printf format %s reads invalid arg 0; indices are 1-based", verb.Raw))
			return
		} else if off != 0 {
			arg, ok := args[off-1].(*ssa.MakeInterface)
			if ok {
				if !checkType(verb.Letter, arg.X.Type(), true) {
					carg.Invalid(fmt.Sprintf("Printf format %s has arg #%d of wrong type %s",
						verb.Raw, ptr, args[ptr-1].(*ssa.MakeInterface).X.Type()))
					return
				}
			}
		}

		switch verb.Value {
		case -1:
			// Consume next argument
			ptr++
		case 0:
			// Don't consume any arguments
		default:
			ptr = verb.Value + 1
		}
	}

	if !hasExplicit && ptr <= len(args) {
		carg.Invalid(fmt.Sprintf("Printf call needs %d args but has %d args", ptr-1, len(args)))
	}
}

func checkAtomicAlignmentImpl(call *Call) {
	sizes := call.Pass.TypesSizes
	if sizes.Sizeof(types.Typ[types.Uintptr]) != 4 {
		// Not running on a 32-bit platform
		return
	}
	v, ok := call.Args[0].Value.Value.(*ssa.FieldAddr)
	if !ok {
		// TODO(dh): also check indexing into arrays and slices
		return
	}
	T := v.X.Type().Underlying().(*types.Pointer).Elem().Underlying().(*types.Struct)
	fields := make([]*types.Var, 0, T.NumFields())
	for i := 0; i < T.NumFields() && i <= v.Field; i++ {
		fields = append(fields, T.Field(i))
	}

	off := sizes.Offsetsof(fields)[v.Field]
	if off%8 != 0 {
		msg := fmt.Sprintf("address of non 64-bit aligned field %s passed to %s",
			T.Field(v.Field).Name(),
			code.CallName(call.Instr.Common()))
		call.Invalid(msg)
	}
}

func checkNoopMarshalImpl(argN int, meths ...string) CallCheck {
	return func(call *Call) {
		if code.IsGenerated(call.Pass, call.Instr.Pos()) {
			return
		}
		arg := call.Args[argN]
		T := arg.Value.Value.Type()
		Ts, ok := code.Dereference(T).Underlying().(*types.Struct)
		if !ok {
			return
		}
		if Ts.NumFields() == 0 {
			return
		}
		fields := code.FlattenFields(Ts)
		for _, field := range fields {
			if field.Var.Exported() {
				return
			}
		}
		// OPT(dh): we could use a method set cache here
		ms := call.Instr.Parent().Prog.MethodSets.MethodSet(T)
		// TODO(dh): we're not checking the signature, which can cause false negatives.
		// This isn't a huge problem, however, since vet complains about incorrect signatures.
		for _, meth := range meths {
			if ms.Lookup(nil, meth) != nil {
				return
			}
		}
		arg.Invalid("struct doesn't have any exported fields, nor custom marshaling")
	}
}

func checkUnsupportedMarshalImpl(argN int, tag string, meths ...string) CallCheck {
	// TODO(dh): flag slices and maps of unsupported types
	return func(call *Call) {
		msCache := &call.Instr.Parent().Prog.MethodSets

		arg := call.Args[argN]
		T := arg.Value.Value.Type()
		Ts, ok := code.Dereference(T).Underlying().(*types.Struct)
		if !ok {
			return
		}
		ms := msCache.MethodSet(T)
		// TODO(dh): we're not checking the signature, which can cause false negatives.
		// This isn't a huge problem, however, since vet complains about incorrect signatures.
		for _, meth := range meths {
			if ms.Lookup(nil, meth) != nil {
				return
			}
		}
		fields := code.FlattenFields(Ts)
		for _, field := range fields {
			if !(field.Var.Exported()) {
				continue
			}
			if reflect.StructTag(field.Tag).Get(tag) == "-" {
				continue
			}
			ms := msCache.MethodSet(field.Var.Type())
			// TODO(dh): we're not checking the signature, which can cause false negatives.
			// This isn't a huge problem, however, since vet complains about incorrect signatures.
			for _, meth := range meths {
				if ms.Lookup(nil, meth) != nil {
					return
				}
			}
			switch field.Var.Type().Underlying().(type) {
			case *types.Chan, *types.Signature:
				arg.Invalid(fmt.Sprintf("trying to marshal chan or func value, field %s", fieldPath(T, field.Path)))
			}
		}
	}
}

func fieldPath(start types.Type, indices []int) string {
	p := start.String()
	for _, idx := range indices {
		field := code.Dereference(start).Underlying().(*types.Struct).Field(idx)
		start = field.Type()
		p += "." + field.Name()
	}
	return p
}

func isInLoop(b *ssa.BasicBlock) bool {
	sets := functions.FindLoops(b.Parent())
	for _, set := range sets {
		if set.Has(b) {
			return true
		}
	}
	return false
}

func CheckUntrappableSignal(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if !code.IsCallToAnyAST(pass, call,
			"os/signal.Ignore", "os/signal.Notify", "os/signal.Reset") {
			return
		}

		hasSigterm := false
		for _, arg := range call.Args {
			if conv, ok := arg.(*ast.CallExpr); ok && isName(pass, conv.Fun, "os.Signal") {
				arg = conv.Args[0]
			}

			if isName(pass, arg, "syscall.SIGTERM") {
				hasSigterm = true
				break
			}

		}
		for i, arg := range call.Args {
			if conv, ok := arg.(*ast.CallExpr); ok && isName(pass, conv.Fun, "os.Signal") {
				arg = conv.Args[0]
			}

			if isName(pass, arg, "os.Kill") || isName(pass, arg, "syscall.SIGKILL") {
				var fixes []analysis.SuggestedFix
				if !hasSigterm {
					nargs := make([]ast.Expr, len(call.Args))
					for j, a := range call.Args {
						if i == j {
							nargs[j] = Selector("syscall", "SIGTERM")
						} else {
							nargs[j] = a
						}
					}
					ncall := *call
					ncall.Args = nargs
					fixes = append(fixes, edit.Fix(fmt.Sprintf("use syscall.SIGTERM instead of %s", report.Render(pass, arg)), edit.ReplaceWithNode(pass.Fset, call, &ncall)))
				}
				nargs := make([]ast.Expr, 0, len(call.Args))
				for j, a := range call.Args {
					if i == j {
						continue
					}
					nargs = append(nargs, a)
				}
				ncall := *call
				ncall.Args = nargs
				fixes = append(fixes, edit.Fix(fmt.Sprintf("remove %s from list of arguments", report.Render(pass, arg)), edit.ReplaceWithNode(pass.Fset, call, &ncall)))
				report.Node(pass, arg, fmt.Sprintf("%s cannot be trapped (did you mean syscall.SIGTERM?)", report.Render(pass, arg)), fixes...)
			}
			if isName(pass, arg, "syscall.SIGSTOP") {
				nargs := make([]ast.Expr, 0, len(call.Args)-1)
				for j, a := range call.Args {
					if i == j {
						continue
					}
					nargs = append(nargs, a)
				}
				ncall := *call
				ncall.Args = nargs
				report.Node(pass, arg, "syscall.SIGSTOP cannot be trapped",
					edit.Fix("remove syscall.SIGSTOP from list of arguments", edit.ReplaceWithNode(pass.Fset, call, &ncall)))
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckTemplate(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		var kind string
		switch code.CallNameAST(pass, call) {
		case "(*text/template.Template).Parse":
			kind = "text"
		case "(*html/template.Template).Parse":
			kind = "html"
		default:
			return
		}
		sel := call.Fun.(*ast.SelectorExpr)
		if !code.IsCallToAnyAST(pass, sel.X, "text/template.New", "html/template.New") {
			// TODO(dh): this is a cheap workaround for templates with
			// different delims. A better solution with less false
			// negatives would use data flow analysis to see where the
			// template comes from and where it has been
			return
		}
		s, ok := code.ExprToString(pass, call.Args[Arg("(*text/template.Template).Parse.text")])
		if !ok {
			return
		}
		var err error
		switch kind {
		case "text":
			_, err = texttemplate.New("").Parse(s)
		case "html":
			_, err = htmltemplate.New("").Parse(s)
		}
		if err != nil {
			// TODO(dominikh): whitelist other parse errors, if any
			if strings.Contains(err.Error(), "unexpected") {
				report.Nodef(pass, call.Args[Arg("(*text/template.Template).Parse.text")], "%s", err)
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkTimeSleepConstantPatternRns = pattern.MustParse(`(BinaryExpr duration "*" (SelectorExpr (Ident "time") (Ident "Nanosecond")))`)
	checkTimeSleepConstantPatternRs  = pattern.MustParse(`(BinaryExpr duration "*" (SelectorExpr (Ident "time") (Ident "Second")))`)
)

func CheckTimeSleepConstant(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if !code.IsCallToAST(pass, call, "time.Sleep") {
			return
		}
		lit, ok := call.Args[Arg("time.Sleep.d")].(*ast.BasicLit)
		if !ok {
			return
		}
		n, err := strconv.Atoi(lit.Value)
		if err != nil {
			return
		}
		if n == 0 || n > 120 {
			// time.Sleep(0) is a seldom used pattern in concurrency
			// tests. >120 might be intentional. 120 was chosen
			// because the user could've meant 2 minutes.
			return
		}

		report.Node(pass, lit,
			fmt.Sprintf("sleeping for %d nanoseconds is probably a bug; be explicit if it isn't", n),
			edit.Fix("explicitly use nanoseconds", edit.ReplaceWithPattern(pass, checkTimeSleepConstantPatternRns, pattern.State{"duration": lit}, lit)),
			edit.Fix("use seconds", edit.ReplaceWithPattern(pass, checkTimeSleepConstantPatternRs, pattern.State{"duration": lit}, lit)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var checkWaitgroupAddQ = pattern.MustParse(`
	(GoStmt
		(CallExpr
			(FuncLit
				_
				call@(CallExpr (Function "(*sync.WaitGroup).Add") _):_) _))`)

func CheckWaitgroupAdd(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkWaitgroupAddQ, node); ok {
			call := m.State["call"].(ast.Node)
			report.Nodef(pass, call, "should call %s before starting the goroutine to avoid a race",
				report.Render(pass, call))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.GoStmt)(nil)}, fn)
	return nil, nil
}

func CheckInfiniteEmptyLoop(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		loop := node.(*ast.ForStmt)
		if len(loop.Body.List) != 0 || loop.Post != nil {
			return
		}

		if loop.Init != nil {
			// TODO(dh): this isn't strictly necessary, it just makes
			// the check easier.
			return
		}
		// An empty loop is bad news in two cases: 1) The loop has no
		// condition. In that case, it's just a loop that spins
		// forever and as fast as it can, keeping a core busy. 2) The
		// loop condition only consists of variable or field reads and
		// operators on those. The only way those could change their
		// value is with unsynchronised access, which constitutes a
		// data race.
		//
		// If the condition contains any function calls, its behaviour
		// is dynamic and the loop might terminate. Similarly for
		// channel receives.

		if loop.Cond != nil {
			if code.MayHaveSideEffects(loop.Cond) {
				return
			}
			if ident, ok := loop.Cond.(*ast.Ident); ok {
				if k, ok := pass.TypesInfo.ObjectOf(ident).(*types.Const); ok {
					if !constant.BoolVal(k.Val()) {
						// don't flag `for false {}` loops. They're a debug aid.
						return
					}
				}
			}
			report.Nodef(pass, loop, "loop condition never changes or has a race condition")
		}
		report.Nodef(pass, loop, "this loop will spin, using 100%% CPU")
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil)}, fn)
	return nil, nil
}

func CheckDeferInInfiniteLoop(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		mightExit := false
		var defers []ast.Stmt
		loop := node.(*ast.ForStmt)
		if loop.Cond != nil {
			return
		}
		fn2 := func(node ast.Node) bool {
			switch stmt := node.(type) {
			case *ast.ReturnStmt:
				mightExit = true
				return false
			case *ast.BranchStmt:
				// TODO(dominikh): if this sees a break in a switch or
				// select, it doesn't check if it breaks the loop or
				// just the select/switch. This causes some false
				// negatives.
				if stmt.Tok == token.BREAK {
					mightExit = true
					return false
				}
			case *ast.DeferStmt:
				defers = append(defers, stmt)
			case *ast.FuncLit:
				// Don't look into function bodies
				return false
			}
			return true
		}
		ast.Inspect(loop.Body, fn2)
		if mightExit {
			return
		}
		for _, stmt := range defers {
			report.Nodef(pass, stmt, "defers in this infinite loop will never run")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil)}, fn)
	return nil, nil
}

func CheckDubiousDeferInChannelRangeLoop(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		loop := node.(*ast.RangeStmt)
		typ := pass.TypesInfo.TypeOf(loop.X)
		_, ok := typ.Underlying().(*types.Chan)
		if !ok {
			return
		}
		fn2 := func(node ast.Node) bool {
			switch stmt := node.(type) {
			case *ast.DeferStmt:
				report.Nodef(pass, stmt, "defers in this range loop won't run unless the channel gets closed")
			case *ast.FuncLit:
				// Don't look into function bodies
				return false
			}
			return true
		}
		ast.Inspect(loop.Body, fn2)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.RangeStmt)(nil)}, fn)
	return nil, nil
}

func CheckTestMainExit(pass *analysis.Pass) (interface{}, error) {
	var (
		fnmain    ast.Node
		callsExit bool
		callsRun  bool
		arg       types.Object
	)
	fn := func(node ast.Node, push bool) bool {
		if !push {
			if fnmain != nil && node == fnmain {
				if !callsExit && callsRun {
					report.Nodef(pass, fnmain, "TestMain should call os.Exit to set exit code")
				}
				fnmain = nil
				callsExit = false
				callsRun = false
				arg = nil
			}
			return true
		}

		switch node := node.(type) {
		case *ast.FuncDecl:
			if fnmain != nil {
				return true
			}
			if !isTestMain(pass, node) {
				return false
			}
			fnmain = node
			arg = pass.TypesInfo.ObjectOf(node.Type.Params.List[0].Names[0])
			return true
		case *ast.CallExpr:
			if code.IsCallToAST(pass, node, "os.Exit") {
				callsExit = true
				return false
			}
			sel, ok := node.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			if arg != pass.TypesInfo.ObjectOf(ident) {
				return true
			}
			if sel.Sel.Name == "Run" {
				callsRun = true
				return false
			}
			return true
		default:
			ExhaustiveTypeSwitch(node)
			return true
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Nodes([]ast.Node{(*ast.FuncDecl)(nil), (*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func isTestMain(pass *analysis.Pass, decl *ast.FuncDecl) bool {
	if decl.Name.Name != "TestMain" {
		return false
	}
	if len(decl.Type.Params.List) != 1 {
		return false
	}
	arg := decl.Type.Params.List[0]
	if len(arg.Names) != 1 {
		return false
	}
	return code.IsOfType(pass, arg.Type, "*testing.M")
}

func CheckExec(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if !code.IsCallToAST(pass, call, "os/exec.Command") {
			return
		}
		val, ok := code.ExprToString(pass, call.Args[Arg("os/exec.Command.name")])
		if !ok {
			return
		}
		if !strings.Contains(val, " ") || strings.Contains(val, `\`) || strings.Contains(val, "/") {
			return
		}
		report.Nodef(pass, call.Args[Arg("os/exec.Command.name")],
			"first argument to exec.Command looks like a shell command, but a program name or path are expected")
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckLoopEmptyDefault(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		loop := node.(*ast.ForStmt)
		if len(loop.Body.List) != 1 || loop.Cond != nil || loop.Init != nil {
			return
		}
		sel, ok := loop.Body.List[0].(*ast.SelectStmt)
		if !ok {
			return
		}
		for _, c := range sel.Body.List {
			// FIXME this leaves behind an empty line, and possibly
			// comments in the default branch. We can't easily fix
			// either.
			if comm, ok := c.(*ast.CommClause); ok && comm.Comm == nil && len(comm.Body) == 0 {
				report.Node(pass, comm, "should not have an empty default case in a for+select loop; the loop will spin",
					edit.Fix("remove empty default branch", edit.Delete(comm)))
				// there can only be one default case
				break
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil)}, fn)
	return nil, nil
}

func CheckLhsRhsIdentical(pass *analysis.Pass) (interface{}, error) {
	// TODO(dh): this check ignores the existence of side-effects and
	// happily flags fn() == fn() – so far, we've had nobody complain
	// about a false positive, and it's caught several bugs in real
	// code.
	fn := func(node ast.Node) {
		op := node.(*ast.BinaryExpr)
		switch op.Op {
		case token.EQL, token.NEQ:
			if basic, ok := pass.TypesInfo.TypeOf(op.X).Underlying().(*types.Basic); ok {
				if kind := basic.Kind(); kind == types.Float32 || kind == types.Float64 {
					// f == f and f != f might be used to check for NaN
					return
				}
			}
		case token.SUB, token.QUO, token.AND, token.REM, token.OR, token.XOR, token.AND_NOT,
			token.LAND, token.LOR, token.LSS, token.GTR, token.LEQ, token.GEQ:
		default:
			// For some ops, such as + and *, it can make sense to
			// have identical operands
			return
		}

		if reflect.TypeOf(op.X) != reflect.TypeOf(op.Y) {
			return
		}
		if report.Render(pass, op.X) != report.Render(pass, op.Y) {
			return
		}
		l1, ok1 := op.X.(*ast.BasicLit)
		l2, ok2 := op.Y.(*ast.BasicLit)
		if ok1 && ok2 && l1.Kind == token.INT && l2.Kind == l1.Kind && l1.Value == "0" && l2.Value == l1.Value && code.IsGenerated(pass, l1.Pos()) {
			// cgo generates the following function call:
			// _cgoCheckPointer(_cgoBase0, 0 == 0) – it uses 0 == 0
			// instead of true in case the user shadowed the
			// identifier. Ideally we'd restrict this exception to
			// calls of _cgoCheckPointer, but it's not worth the
			// hassle of keeping track of the stack. <lit> <op> <lit>
			// are very rare to begin with, and we're mostly checking
			// for them to catch typos such as 1 == 1 where the user
			// meant to type i == 1. The odds of a false negative for
			// 0 == 0 are slim.
			return
		}
		report.Nodef(pass, op, "identical expressions on the left and right side of the '%s' operator", op.Op)
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func CheckScopedBreak(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		var body *ast.BlockStmt
		switch node := node.(type) {
		case *ast.ForStmt:
			body = node.Body
		case *ast.RangeStmt:
			body = node.Body
		default:
			ExhaustiveTypeSwitch(node)
		}
		for _, stmt := range body.List {
			var blocks [][]ast.Stmt
			switch stmt := stmt.(type) {
			case *ast.SwitchStmt:
				for _, c := range stmt.Body.List {
					blocks = append(blocks, c.(*ast.CaseClause).Body)
				}
			case *ast.SelectStmt:
				for _, c := range stmt.Body.List {
					blocks = append(blocks, c.(*ast.CommClause).Body)
				}
			default:
				continue
			}

			for _, body := range blocks {
				if len(body) == 0 {
					continue
				}
				lasts := []ast.Stmt{body[len(body)-1]}
				// TODO(dh): unfold all levels of nested block
				// statements, not just a single level if statement
				if ifs, ok := lasts[0].(*ast.IfStmt); ok {
					if len(ifs.Body.List) == 0 {
						continue
					}
					lasts[0] = ifs.Body.List[len(ifs.Body.List)-1]

					if block, ok := ifs.Else.(*ast.BlockStmt); ok {
						if len(block.List) != 0 {
							lasts = append(lasts, block.List[len(block.List)-1])
						}
					}
				}
				for _, last := range lasts {
					branch, ok := last.(*ast.BranchStmt)
					if !ok || branch.Tok != token.BREAK || branch.Label != nil {
						continue
					}
					report.Nodef(pass, branch, "ineffective break statement. Did you mean to break out of the outer loop?")
				}
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ForStmt)(nil), (*ast.RangeStmt)(nil)}, fn)
	return nil, nil
}

func CheckUnsafePrintf(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		name := code.CallNameAST(pass, call)
		var arg int

		switch name {
		case "fmt.Printf", "fmt.Sprintf", "log.Printf":
			arg = Arg("fmt.Printf.format")
		case "fmt.Fprintf":
			arg = Arg("fmt.Fprintf.format")
		default:
			return
		}
		if len(call.Args) != arg+1 {
			return
		}
		switch call.Args[arg].(type) {
		case *ast.CallExpr, *ast.Ident:
		default:
			return
		}

		alt := name[:len(name)-1]
		report.Node(pass, call,
			"printf-style function with dynamic format string and no further arguments should use print-style function instead",
			edit.Fix(fmt.Sprintf("use %s instead of %s", alt, name), edit.ReplaceWithString(pass.Fset, call.Fun, alt)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckEarlyDefer(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		block := node.(*ast.BlockStmt)
		if len(block.List) < 2 {
			return
		}
		for i, stmt := range block.List {
			if i == len(block.List)-1 {
				break
			}
			assign, ok := stmt.(*ast.AssignStmt)
			if !ok {
				continue
			}
			if len(assign.Rhs) != 1 {
				continue
			}
			if len(assign.Lhs) < 2 {
				continue
			}
			if lhs, ok := assign.Lhs[len(assign.Lhs)-1].(*ast.Ident); ok && lhs.Name == "_" {
				continue
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			sig, ok := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
			if !ok {
				continue
			}
			if sig.Results().Len() < 2 {
				continue
			}
			last := sig.Results().At(sig.Results().Len() - 1)
			// FIXME(dh): check that it's error from universe, not
			// another type of the same name
			if last.Type().String() != "error" {
				continue
			}
			lhs, ok := assign.Lhs[0].(*ast.Ident)
			if !ok {
				continue
			}
			def, ok := block.List[i+1].(*ast.DeferStmt)
			if !ok {
				continue
			}
			sel, ok := def.Call.Fun.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			ident, ok := selectorX(sel).(*ast.Ident)
			if !ok {
				continue
			}
			if ident.Obj != lhs.Obj {
				continue
			}
			if sel.Sel.Name != "Close" {
				continue
			}
			report.Nodef(pass, def, "should check returned error before deferring %s", report.Render(pass, def.Call))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BlockStmt)(nil)}, fn)
	return nil, nil
}

func selectorX(sel *ast.SelectorExpr) ast.Node {
	switch x := sel.X.(type) {
	case *ast.SelectorExpr:
		return selectorX(x)
	default:
		return x
	}
}

func CheckEmptyCriticalSection(pass *analysis.Pass) (interface{}, error) {
	// Initially it might seem like this check would be easier to
	// implement in SSA. After all, we're only checking for two
	// consecutive method calls. In reality, however, there may be any
	// number of other instructions between the lock and unlock, while
	// still constituting an empty critical section. For example,
	// given `m.x().Lock(); m.x().Unlock()`, there will be a call to
	// x(). In the AST-based approach, this has a tiny potential for a
	// false positive (the second call to x might be doing work that
	// is protected by the mutex). In an SSA-based approach, however,
	// it would miss a lot of real bugs.

	mutexParams := func(s ast.Stmt) (x ast.Expr, funcName string, ok bool) {
		expr, ok := s.(*ast.ExprStmt)
		if !ok {
			return nil, "", false
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			return nil, "", false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return nil, "", false
		}

		fn, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Func)
		if !ok {
			return nil, "", false
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 0 || sig.Results().Len() != 0 {
			return nil, "", false
		}

		return sel.X, fn.Name(), true
	}

	fn := func(node ast.Node) {
		block := node.(*ast.BlockStmt)
		if len(block.List) < 2 {
			return
		}
		for i := range block.List[:len(block.List)-1] {
			sel1, method1, ok1 := mutexParams(block.List[i])
			sel2, method2, ok2 := mutexParams(block.List[i+1])

			if !ok1 || !ok2 || report.Render(pass, sel1) != report.Render(pass, sel2) {
				continue
			}
			if (method1 == "Lock" && method2 == "Unlock") ||
				(method1 == "RLock" && method2 == "RUnlock") {
				report.Nodef(pass, block.List[i+1], "empty critical section")
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BlockStmt)(nil)}, fn)
	return nil, nil
}

var (
	// cgo produces code like fn(&*_Cvar_kSomeCallbacks) which we don't
	// want to flag.
	cgoIdent               = regexp.MustCompile(`^_C(func|var)_.+$`)
	checkIneffectiveCopyQ1 = pattern.MustParse(`(UnaryExpr "&" (StarExpr obj))`)
	checkIneffectiveCopyQ2 = pattern.MustParse(`(StarExpr (UnaryExpr "&" _))`)
)

func CheckIneffectiveCopy(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkIneffectiveCopyQ1, node); ok {
			if ident, ok := m.State["obj"].(*ast.Ident); !ok || !cgoIdent.MatchString(ident.Name) {
				report.Nodef(pass, node, "&*x will be simplified to x. It will not copy x.")
			}
		} else if _, ok := Match(pass, checkIneffectiveCopyQ2, node); ok {
			report.Nodef(pass, node, "*&x will be simplified to x. It will not copy x.")
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.UnaryExpr)(nil), (*ast.StarExpr)(nil)}, fn)
	return nil, nil
}

func CheckDiffSizeComparison(pass *analysis.Pass) (interface{}, error) {
	ranges := pass.ResultOf[valueRangesAnalyzer].(map[*ssa.Function]vrp.Ranges)
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, b := range ssafn.Blocks {
			for _, ins := range b.Instrs {
				binop, ok := ins.(*ssa.BinOp)
				if !ok {
					continue
				}
				if binop.Op != token.EQL && binop.Op != token.NEQ {
					continue
				}
				_, ok1 := binop.X.(*ssa.Slice)
				_, ok2 := binop.Y.(*ssa.Slice)
				if !ok1 && !ok2 {
					continue
				}
				r := ranges[ssafn]
				r1, ok1 := r.Get(binop.X).(vrp.StringInterval)
				r2, ok2 := r.Get(binop.Y).(vrp.StringInterval)
				if !ok1 || !ok2 {
					continue
				}
				if r1.Length.Intersection(r2.Length).Empty() {
					pass.Reportf(binop.Pos(), "comparing strings of different sizes for equality will always return false")
				}
			}
		}
	}
	return nil, nil
}

func CheckCanonicalHeaderKey(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node, push bool) bool {
		if !push {
			return false
		}
		assign, ok := node.(*ast.AssignStmt)
		if ok {
			// TODO(dh): This risks missing some Header reads, for
			// example in `h1["foo"] = h2["foo"]` – these edge
			// cases are probably rare enough to ignore for now.
			for _, expr := range assign.Lhs {
				op, ok := expr.(*ast.IndexExpr)
				if !ok {
					continue
				}
				if code.IsOfType(pass, op.X, "net/http.Header") {
					return false
				}
			}
			return true
		}
		op, ok := node.(*ast.IndexExpr)
		if !ok {
			return true
		}
		if !code.IsOfType(pass, op.X, "net/http.Header") {
			return true
		}
		s, ok := code.ExprToString(pass, op.Index)
		if !ok {
			return true
		}
		canonical := http.CanonicalHeaderKey(s)
		if s == canonical {
			return true
		}
		var fix analysis.SuggestedFix
		switch op.Index.(type) {
		case *ast.BasicLit:
			fix = edit.Fix("canonicalize header key", edit.ReplaceWithString(pass.Fset, op.Index, strconv.Quote(canonical)))
		case *ast.Ident:
			call := &ast.CallExpr{
				Fun:  Selector("http", "CanonicalHeaderKey"),
				Args: []ast.Expr{op.Index},
			}
			fix = edit.Fix("wrap in http.CanonicalHeaderKey", edit.ReplaceWithNode(pass.Fset, op.Index, call))
		}
		msg := fmt.Sprintf("keys in http.Header are canonicalized, %q is not canonical; fix the constant or use http.CanonicalHeaderKey", s)
		if fix.Message != "" {
			report.Node(pass, op, msg, fix)
		} else {
			report.Node(pass, op, msg)
		}
		return true
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Nodes([]ast.Node{(*ast.AssignStmt)(nil), (*ast.IndexExpr)(nil)}, fn)
	return nil, nil
}

func CheckBenchmarkN(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		assign := node.(*ast.AssignStmt)
		if len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return
		}
		sel, ok := assign.Lhs[0].(*ast.SelectorExpr)
		if !ok {
			return
		}
		if sel.Sel.Name != "N" {
			return
		}
		if !code.IsOfType(pass, sel.X, "*testing.B") {
			return
		}
		report.Nodef(pass, assign, "should not assign to %s", report.Render(pass, sel))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.AssignStmt)(nil)}, fn)
	return nil, nil
}

func CheckUnreadVariableValues(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		if code.IsExample(ssafn) {
			continue
		}
		node := ssafn.Syntax()
		if node == nil {
			continue
		}
		if gen, ok := code.Generator(pass, node.Pos()); ok && gen == facts.Goyacc {
			// Don't flag unused values in code generated by goyacc.
			// There may be hundreds of those due to the way the state
			// machine is constructed.
			continue
		}

		switchTags := map[ssa.Value]struct{}{}
		ast.Inspect(node, func(node ast.Node) bool {
			s, ok := node.(*ast.SwitchStmt)
			if !ok {
				return true
			}
			v, _ := ssafn.ValueForExpr(s.Tag)
			switchTags[v] = struct{}{}
			return true
		})

		hasUse := func(v ssa.Value) bool {
			if _, ok := switchTags[v]; ok {
				return true
			}
			refs := v.Referrers()
			if refs == nil {
				// TODO investigate why refs can be nil
				return true
			}
			return len(code.FilterDebug(*refs)) > 0
		}

		ast.Inspect(node, func(node ast.Node) bool {
			assign, ok := node.(*ast.AssignStmt)
			if !ok {
				return true
			}
			if len(assign.Lhs) > 1 && len(assign.Rhs) == 1 {
				// Either a function call with multiple return values,
				// or a comma-ok assignment

				val, _ := ssafn.ValueForExpr(assign.Rhs[0])
				if val == nil {
					return true
				}
				refs := val.Referrers()
				if refs == nil {
					return true
				}
				for _, ref := range *refs {
					ex, ok := ref.(*ssa.Extract)
					if !ok {
						continue
					}
					if !hasUse(ex) {
						lhs := assign.Lhs[ex.Index]
						if ident, ok := lhs.(*ast.Ident); !ok || ok && ident.Name == "_" {
							continue
						}
						report.Nodef(pass, lhs, "this value of %s is never used", lhs)
					}
				}
				return true
			}
			for i, lhs := range assign.Lhs {
				rhs := assign.Rhs[i]
				if ident, ok := lhs.(*ast.Ident); !ok || ok && ident.Name == "_" {
					continue
				}
				val, _ := ssafn.ValueForExpr(rhs)
				if val == nil {
					continue
				}

				if !hasUse(val) {
					report.Nodef(pass, lhs, "this value of %s is never used", lhs)
				}
			}
			return true
		})
	}
	return nil, nil
}

func CheckPredeterminedBooleanExprs(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				ssabinop, ok := ins.(*ssa.BinOp)
				if !ok {
					continue
				}
				switch ssabinop.Op {
				case token.GTR, token.LSS, token.EQL, token.NEQ, token.LEQ, token.GEQ:
				default:
					continue
				}

				xs, ok1 := consts(ssabinop.X, nil, nil)
				ys, ok2 := consts(ssabinop.Y, nil, nil)
				if !ok1 || !ok2 || len(xs) == 0 || len(ys) == 0 {
					continue
				}

				trues := 0
				for _, x := range xs {
					for _, y := range ys {
						if x.Value == nil {
							if y.Value == nil {
								trues++
							}
							continue
						}
						if constant.Compare(x.Value, ssabinop.Op, y.Value) {
							trues++
						}
					}
				}
				b := trues != 0
				if trues == 0 || trues == len(xs)*len(ys) {
					pass.Reportf(ssabinop.Pos(), "binary expression is always %t for all possible values (%s %s %s)",
						b, xs, ssabinop.Op, ys)
				}
			}
		}
	}
	return nil, nil
}

func CheckNilMaps(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				mu, ok := ins.(*ssa.MapUpdate)
				if !ok {
					continue
				}
				c, ok := mu.Map.(*ssa.Const)
				if !ok {
					continue
				}
				if c.Value != nil {
					continue
				}
				pass.Reportf(mu.Pos(), "assignment to nil map")
			}
		}
	}
	return nil, nil
}

func CheckExtremeComparison(pass *analysis.Pass) (interface{}, error) {
	isobj := func(expr ast.Expr, name string) bool {
		sel, ok := expr.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		return code.IsObject(pass.TypesInfo.ObjectOf(sel.Sel), name)
	}

	fn := func(node ast.Node) {
		expr := node.(*ast.BinaryExpr)
		tx := pass.TypesInfo.TypeOf(expr.X)
		basic, ok := tx.Underlying().(*types.Basic)
		if !ok {
			return
		}

		var max string
		var min string

		switch basic.Kind() {
		case types.Uint8:
			max = "math.MaxUint8"
		case types.Uint16:
			max = "math.MaxUint16"
		case types.Uint32:
			max = "math.MaxUint32"
		case types.Uint64:
			max = "math.MaxUint64"
		case types.Uint:
			max = "math.MaxUint64"

		case types.Int8:
			min = "math.MinInt8"
			max = "math.MaxInt8"
		case types.Int16:
			min = "math.MinInt16"
			max = "math.MaxInt16"
		case types.Int32:
			min = "math.MinInt32"
			max = "math.MaxInt32"
		case types.Int64:
			min = "math.MinInt64"
			max = "math.MaxInt64"
		case types.Int:
			min = "math.MinInt64"
			max = "math.MaxInt64"
		}

		if (expr.Op == token.GTR || expr.Op == token.GEQ) && isobj(expr.Y, max) ||
			(expr.Op == token.LSS || expr.Op == token.LEQ) && isobj(expr.X, max) {
			report.Nodef(pass, expr, "no value of type %s is greater than %s", basic, max)
		}
		if expr.Op == token.LEQ && isobj(expr.Y, max) ||
			expr.Op == token.GEQ && isobj(expr.X, max) {
			report.Nodef(pass, expr, "every value of type %s is <= %s", basic, max)
		}

		if (basic.Info() & types.IsUnsigned) != 0 {
			if (expr.Op == token.LSS && code.IsIntLiteral(expr.Y, "0")) ||
				(expr.Op == token.GTR && code.IsIntLiteral(expr.X, "0")) {
				report.Nodef(pass, expr, "no value of type %s is less than 0", basic)
			}
			if expr.Op == token.GEQ && code.IsIntLiteral(expr.Y, "0") ||
				expr.Op == token.LEQ && code.IsIntLiteral(expr.X, "0") {
				report.Nodef(pass, expr, "every value of type %s is >= 0", basic)
			}
		} else {
			if (expr.Op == token.LSS || expr.Op == token.LEQ) && isobj(expr.Y, min) ||
				(expr.Op == token.GTR || expr.Op == token.GEQ) && isobj(expr.X, min) {
				report.Nodef(pass, expr, "no value of type %s is less than %s", basic, min)
			}
			if expr.Op == token.GEQ && isobj(expr.Y, min) ||
				expr.Op == token.LEQ && isobj(expr.X, min) {
				report.Nodef(pass, expr, "every value of type %s is >= %s", basic, min)
			}
		}

	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func consts(val ssa.Value, out []*ssa.Const, visitedPhis map[string]bool) ([]*ssa.Const, bool) {
	if visitedPhis == nil {
		visitedPhis = map[string]bool{}
	}
	var ok bool
	switch val := val.(type) {
	case *ssa.Phi:
		if visitedPhis[val.Name()] {
			break
		}
		visitedPhis[val.Name()] = true
		vals := val.Operands(nil)
		for _, phival := range vals {
			out, ok = consts(*phival, out, visitedPhis)
			if !ok {
				return nil, false
			}
		}
	case *ssa.Const:
		out = append(out, val)
	case *ssa.Convert:
		out, ok = consts(val.X, out, visitedPhis)
		if !ok {
			return nil, false
		}
	default:
		return nil, false
	}
	if len(out) < 2 {
		return out, true
	}
	uniq := []*ssa.Const{out[0]}
	for _, val := range out[1:] {
		if val.Value == uniq[len(uniq)-1].Value {
			continue
		}
		uniq = append(uniq, val)
	}
	return uniq, true
}

func CheckLoopCondition(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		fn := func(node ast.Node) bool {
			loop, ok := node.(*ast.ForStmt)
			if !ok {
				return true
			}
			if loop.Init == nil || loop.Cond == nil || loop.Post == nil {
				return true
			}
			init, ok := loop.Init.(*ast.AssignStmt)
			if !ok || len(init.Lhs) != 1 || len(init.Rhs) != 1 {
				return true
			}
			cond, ok := loop.Cond.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			x, ok := cond.X.(*ast.Ident)
			if !ok {
				return true
			}
			lhs, ok := init.Lhs[0].(*ast.Ident)
			if !ok {
				return true
			}
			if x.Obj != lhs.Obj {
				return true
			}
			if _, ok := loop.Post.(*ast.IncDecStmt); !ok {
				return true
			}

			v, isAddr := ssafn.ValueForExpr(cond.X)
			if v == nil || isAddr {
				return true
			}
			switch v := v.(type) {
			case *ssa.Phi:
				ops := v.Operands(nil)
				if len(ops) != 2 {
					return true
				}
				_, ok := (*ops[0]).(*ssa.Const)
				if !ok {
					return true
				}
				sigma, ok := (*ops[1]).(*ssa.Sigma)
				if !ok {
					return true
				}
				if sigma.X != v {
					return true
				}
			case *ssa.Load:
				return true
			}
			report.Nodef(pass, cond, "variable in loop condition never changes")

			return true
		}
		Inspect(ssafn.Syntax(), fn)
	}
	return nil, nil
}

func CheckArgOverwritten(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		fn := func(node ast.Node) bool {
			var typ *ast.FuncType
			var body *ast.BlockStmt
			switch fn := node.(type) {
			case *ast.FuncDecl:
				typ = fn.Type
				body = fn.Body
			case *ast.FuncLit:
				typ = fn.Type
				body = fn.Body
			}
			if body == nil {
				return true
			}
			if len(typ.Params.List) == 0 {
				return true
			}
			for _, field := range typ.Params.List {
				for _, arg := range field.Names {
					obj := pass.TypesInfo.ObjectOf(arg)
					var ssaobj *ssa.Parameter
					for _, param := range ssafn.Params {
						if param.Object() == obj {
							ssaobj = param
							break
						}
					}
					if ssaobj == nil {
						continue
					}
					refs := ssaobj.Referrers()
					if refs == nil {
						continue
					}
					if len(code.FilterDebug(*refs)) != 0 {
						continue
					}

					assigned := false
					ast.Inspect(body, func(node ast.Node) bool {
						assign, ok := node.(*ast.AssignStmt)
						if !ok {
							return true
						}
						for _, lhs := range assign.Lhs {
							ident, ok := lhs.(*ast.Ident)
							if !ok {
								continue
							}
							if pass.TypesInfo.ObjectOf(ident) == obj {
								assigned = true
								return false
							}
						}
						return true
					})
					if assigned {
						report.Nodef(pass, arg, "argument %s is overwritten before first use", arg)
					}
				}
			}
			return true
		}
		Inspect(ssafn.Syntax(), fn)
	}
	return nil, nil
}

func CheckIneffectiveLoop(pass *analysis.Pass) (interface{}, error) {
	// This check detects some, but not all unconditional loop exits.
	// We give up in the following cases:
	//
	// - a goto anywhere in the loop. The goto might skip over our
	// return, and we don't check that it doesn't.
	//
	// - any nested, unlabelled continue, even if it is in another
	// loop or closure.
	fn := func(node ast.Node) {
		var body *ast.BlockStmt
		switch fn := node.(type) {
		case *ast.FuncDecl:
			body = fn.Body
		case *ast.FuncLit:
			body = fn.Body
		default:
			ExhaustiveTypeSwitch(node)
		}
		if body == nil {
			return
		}
		labels := map[*ast.Object]ast.Stmt{}
		ast.Inspect(body, func(node ast.Node) bool {
			label, ok := node.(*ast.LabeledStmt)
			if !ok {
				return true
			}
			labels[label.Label.Obj] = label.Stmt
			return true
		})

		ast.Inspect(body, func(node ast.Node) bool {
			var loop ast.Node
			var body *ast.BlockStmt
			switch node := node.(type) {
			case *ast.ForStmt:
				body = node.Body
				loop = node
			case *ast.RangeStmt:
				typ := pass.TypesInfo.TypeOf(node.X)
				if _, ok := typ.Underlying().(*types.Map); ok {
					// looping once over a map is a valid pattern for
					// getting an arbitrary element.
					return true
				}
				body = node.Body
				loop = node
			default:
				return true
			}
			if len(body.List) < 2 {
				// avoid flagging the somewhat common pattern of using
				// a range loop to get the first element in a slice,
				// or the first rune in a string.
				return true
			}
			var unconditionalExit ast.Node
			hasBranching := false
			for _, stmt := range body.List {
				switch stmt := stmt.(type) {
				case *ast.BranchStmt:
					switch stmt.Tok {
					case token.BREAK:
						if stmt.Label == nil || labels[stmt.Label.Obj] == loop {
							unconditionalExit = stmt
						}
					case token.CONTINUE:
						if stmt.Label == nil || labels[stmt.Label.Obj] == loop {
							unconditionalExit = nil
							return false
						}
					}
				case *ast.ReturnStmt:
					unconditionalExit = stmt
				case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.SelectStmt:
					hasBranching = true
				}
			}
			if unconditionalExit == nil || !hasBranching {
				return false
			}
			ast.Inspect(body, func(node ast.Node) bool {
				if branch, ok := node.(*ast.BranchStmt); ok {

					switch branch.Tok {
					case token.GOTO:
						unconditionalExit = nil
						return false
					case token.CONTINUE:
						if branch.Label != nil && labels[branch.Label.Obj] != loop {
							return true
						}
						unconditionalExit = nil
						return false
					}
				}
				return true
			})
			if unconditionalExit != nil {
				report.Nodef(pass, unconditionalExit, "the surrounding loop is unconditionally terminated")
			}
			return true
		})
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.FuncDecl)(nil), (*ast.FuncLit)(nil)}, fn)
	return nil, nil
}

var checkNilContextQ = pattern.MustParse(`(CallExpr fun@(Function _) (Builtin "nil"):_)`)

func CheckNilContext(pass *analysis.Pass) (interface{}, error) {
	todo := &ast.CallExpr{
		Fun: Selector("context", "TODO"),
	}
	bg := &ast.CallExpr{
		Fun: Selector("context", "Background"),
	}
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkNilContextQ, node)
		if !ok {
			return
		}

		call := node.(*ast.CallExpr)
		fun, ok := m.State["fun"].(*types.Func)
		if !ok {
			// it might also be a builtin
			return
		}
		sig := fun.Type().(*types.Signature)
		if sig.Params().Len() == 0 {
			// Our CallExpr might've matched a method expression, like
			// (*T).Foo(nil) – here, nil isn't the first argument of
			// the Foo method, but the method receiver.
			return
		}
		if !code.IsType(sig.Params().At(0).Type(), "context.Context") {
			return
		}
		report.Node(pass, call.Args[0],
			"do not pass a nil Context, even if a function permits it; pass context.TODO if you are unsure about which Context to use",
			edit.Fix("use context.TODO", edit.ReplaceWithNode(pass.Fset, call.Args[0], todo)),
			edit.Fix("use context.Background", edit.ReplaceWithNode(pass.Fset, call.Args[0], bg)))
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

var (
	checkSeekerQ = pattern.MustParse(`(CallExpr fun@(SelectorExpr _ (Ident "Seek")) [arg1@(SelectorExpr (Ident "io") (Ident (Or "SeekStart" "SeekCurrent" "SeekEnd"))) arg2])`)
	checkSeekerR = pattern.MustParse(`(CallExpr fun [arg2 arg1])`)
)

func CheckSeeker(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if _, edits, ok := MatchAndEdit(pass, checkSeekerQ, checkSeekerR, node); ok {
			report.Node(pass, node, "the first argument of io.Seeker is the offset, but an io.Seek* constant is being used instead",
				edit.Fix("swap arguments", edits...))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckIneffectiveAppend(pass *analysis.Pass) (interface{}, error) {
	isAppend := func(ins ssa.Value) bool {
		call, ok := ins.(*ssa.Call)
		if !ok {
			return false
		}
		if call.Call.IsInvoke() {
			return false
		}
		if builtin, ok := call.Call.Value.(*ssa.Builtin); !ok || builtin.Name() != "append" {
			return false
		}
		return true
	}

	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				val, ok := ins.(ssa.Value)
				if !ok || !isAppend(val) {
					continue
				}

				isUsed := false
				visited := map[ssa.Instruction]bool{}
				var walkRefs func(refs []ssa.Instruction)
				walkRefs = func(refs []ssa.Instruction) {
				loop:
					for _, ref := range refs {
						if visited[ref] {
							continue
						}
						visited[ref] = true
						if _, ok := ref.(*ssa.DebugRef); ok {
							continue
						}
						switch ref := ref.(type) {
						case *ssa.Phi:
							walkRefs(*ref.Referrers())
						case *ssa.Sigma:
							walkRefs(*ref.Referrers())
						case ssa.Value:
							if !isAppend(ref) {
								isUsed = true
							} else {
								walkRefs(*ref.Referrers())
							}
						case ssa.Instruction:
							isUsed = true
							break loop
						}
					}
				}

				refs := val.Referrers()
				if refs == nil {
					continue
				}
				walkRefs(*refs)

				if !isUsed {
					pass.Reportf(ins.Pos(), "this result of append is never used, except maybe in other appends")
				}
			}
		}
	}
	return nil, nil
}

func CheckConcurrentTesting(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				gostmt, ok := ins.(*ssa.Go)
				if !ok {
					continue
				}
				var fn *ssa.Function
				switch val := gostmt.Call.Value.(type) {
				case *ssa.Function:
					fn = val
				case *ssa.MakeClosure:
					fn = val.Fn.(*ssa.Function)
				default:
					continue
				}
				if fn.Blocks == nil {
					continue
				}
				for _, block := range fn.Blocks {
					for _, ins := range block.Instrs {
						call, ok := ins.(*ssa.Call)
						if !ok {
							continue
						}
						if call.Call.IsInvoke() {
							continue
						}
						callee := call.Call.StaticCallee()
						if callee == nil {
							continue
						}
						recv := callee.Signature.Recv()
						if recv == nil {
							continue
						}
						if !code.IsType(recv.Type(), "*testing.common") {
							continue
						}
						fn, ok := call.Call.StaticCallee().Object().(*types.Func)
						if !ok {
							continue
						}
						name := fn.Name()
						switch name {
						case "FailNow", "Fatal", "Fatalf", "SkipNow", "Skip", "Skipf":
						default:
							continue
						}
						pass.Reportf(gostmt.Pos(), "the goroutine calls T.%s, which must be called in the same goroutine as the test", name)
					}
				}
			}
		}
	}
	return nil, nil
}

func eachCall(ssafn *ssa.Function, fn func(caller *ssa.Function, site ssa.CallInstruction, callee *ssa.Function)) {
	for _, b := range ssafn.Blocks {
		for _, instr := range b.Instrs {
			if site, ok := instr.(ssa.CallInstruction); ok {
				if g := site.Common().StaticCallee(); g != nil {
					fn(ssafn, site, g)
				}
			}
		}
	}
}

func CheckCyclicFinalizer(pass *analysis.Pass) (interface{}, error) {
	fn := func(caller *ssa.Function, site ssa.CallInstruction, callee *ssa.Function) {
		if callee.RelString(nil) != "runtime.SetFinalizer" {
			return
		}
		arg0 := site.Common().Args[Arg("runtime.SetFinalizer.obj")]
		if iface, ok := arg0.(*ssa.MakeInterface); ok {
			arg0 = iface.X
		}
		load, ok := arg0.(*ssa.Load)
		if !ok {
			return
		}
		v, ok := load.X.(*ssa.Alloc)
		if !ok {
			return
		}
		arg1 := site.Common().Args[Arg("runtime.SetFinalizer.finalizer")]
		if iface, ok := arg1.(*ssa.MakeInterface); ok {
			arg1 = iface.X
		}
		mc, ok := arg1.(*ssa.MakeClosure)
		if !ok {
			return
		}
		for _, b := range mc.Bindings {
			if b == v {
				pos := lint.DisplayPosition(pass.Fset, mc.Fn.Pos())
				pass.Reportf(site.Pos(), "the finalizer closes over the object, preventing the finalizer from ever running (at %s)", pos)
			}
		}
	}
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		eachCall(ssafn, fn)
	}
	return nil, nil
}

/*
func CheckSliceOutOfBounds(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				ia, ok := ins.(*ssa.IndexAddr)
				if !ok {
					continue
				}
				if _, ok := ia.X.Type().Underlying().(*types.Slice); !ok {
					continue
				}
				sr, ok1 := c.funcDescs.Get(ssafn).Ranges[ia.X].(vrp.SliceInterval)
				idxr, ok2 := c.funcDescs.Get(ssafn).Ranges[ia.Index].(vrp.IntInterval)
				if !ok1 || !ok2 || !sr.IsKnown() || !idxr.IsKnown() || sr.Length.Empty() || idxr.Empty() {
					continue
				}
				if idxr.Lower.Cmp(sr.Length.Upper) >= 0 {
					report.Nodef(pass, ia, "index out of bounds")
				}
			}
		}
	}
	return nil, nil
}
*/

func CheckDeferLock(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range ssafn.Blocks {
			instrs := code.FilterDebug(block.Instrs)
			if len(instrs) < 2 {
				continue
			}
			for i, ins := range instrs[:len(instrs)-1] {
				call, ok := ins.(*ssa.Call)
				if !ok {
					continue
				}
				if !code.IsCallToAny(call.Common(), "(*sync.Mutex).Lock", "(*sync.RWMutex).RLock") {
					continue
				}
				nins, ok := instrs[i+1].(*ssa.Defer)
				if !ok {
					continue
				}
				if !code.IsCallToAny(&nins.Call, "(*sync.Mutex).Lock", "(*sync.RWMutex).RLock") {
					continue
				}
				if call.Common().Args[0] != nins.Call.Args[0] {
					continue
				}
				name := shortCallName(call.Common())
				alt := ""
				switch name {
				case "Lock":
					alt = "Unlock"
				case "RLock":
					alt = "RUnlock"
				}
				pass.Reportf(nins.Pos(), "deferring %s right after having locked already; did you mean to defer %s?", name, alt)
			}
		}
	}
	return nil, nil
}

func CheckNaNComparison(pass *analysis.Pass) (interface{}, error) {
	isNaN := func(v ssa.Value) bool {
		call, ok := v.(*ssa.Call)
		if !ok {
			return false
		}
		return code.IsCallTo(call.Common(), "math.NaN")
	}
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				ins, ok := ins.(*ssa.BinOp)
				if !ok {
					continue
				}
				if isNaN(ins.X) || isNaN(ins.Y) {
					pass.Reportf(ins.Pos(), "no value is equal to NaN, not even NaN itself")
				}
			}
		}
	}
	return nil, nil
}

func CheckInfiniteRecursion(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		eachCall(ssafn, func(caller *ssa.Function, site ssa.CallInstruction, callee *ssa.Function) {
			if callee != ssafn {
				return
			}
			if _, ok := site.(*ssa.Go); ok {
				// Recursively spawning goroutines doesn't consume
				// stack space infinitely, so don't flag it.
				return
			}

			block := site.Block()
			canReturn := false
			for _, b := range ssafn.Blocks {
				if block.Dominates(b) {
					continue
				}
				if len(b.Instrs) == 0 {
					continue
				}
				if _, ok := b.Control().(*ssa.Return); ok {
					canReturn = true
					break
				}
			}
			if canReturn {
				return
			}
			pass.Reportf(site.Pos(), "infinite recursive call")
		})
	}
	return nil, nil
}

func objectName(obj types.Object) string {
	if obj == nil {
		return "<nil>"
	}
	var name string
	if obj.Pkg() != nil && obj.Pkg().Scope().Lookup(obj.Name()) == obj {
		s := obj.Pkg().Path()
		if s != "" {
			name += s + "."
		}
	}
	name += obj.Name()
	return name
}

func isName(pass *analysis.Pass, expr ast.Expr, name string) bool {
	var obj types.Object
	switch expr := expr.(type) {
	case *ast.Ident:
		obj = pass.TypesInfo.ObjectOf(expr)
	case *ast.SelectorExpr:
		obj = pass.TypesInfo.ObjectOf(expr.Sel)
	}
	return objectName(obj) == name
}

func CheckLeakyTimeTick(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		if code.IsMainLike(pass) || code.IsInTest(pass, ssafn) {
			continue
		}
		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				call, ok := ins.(*ssa.Call)
				if !ok || !code.IsCallTo(call.Common(), "time.Tick") {
					continue
				}
				if !functions.Terminates(call.Parent()) {
					continue
				}
				pass.Reportf(call.Pos(), "using time.Tick leaks the underlying ticker, consider using it only in endless functions, tests and the main package, and use time.NewTicker here")
			}
		}
	}
	return nil, nil
}

var checkDoubleNegationQ = pattern.MustParse(`(UnaryExpr "!" single@(UnaryExpr "!" x))`)

func CheckDoubleNegation(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		if m, ok := Match(pass, checkDoubleNegationQ, node); ok {
			report.Node(pass, node, "negating a boolean twice has no effect; is this a typo?",
				edit.Fix("turn into single negation", edit.ReplaceWithNode(pass.Fset, node, m.State["single"].(ast.Node))),
				edit.Fix("remove double negation", edit.ReplaceWithNode(pass.Fset, node, m.State["x"].(ast.Node))))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.UnaryExpr)(nil)}, fn)
	return nil, nil
}

func CheckRepeatedIfElse(pass *analysis.Pass) (interface{}, error) {
	seen := map[ast.Node]bool{}

	var collectConds func(ifstmt *ast.IfStmt, conds []ast.Expr) ([]ast.Expr, bool)
	collectConds = func(ifstmt *ast.IfStmt, conds []ast.Expr) ([]ast.Expr, bool) {
		seen[ifstmt] = true
		// Bail if any if-statement has an Init statement or side effects in its condition
		if ifstmt.Init != nil {
			return nil, false
		}
		if code.MayHaveSideEffects(ifstmt.Cond) {
			return nil, false
		}

		conds = append(conds, ifstmt.Cond)
		if elsestmt, ok := ifstmt.Else.(*ast.IfStmt); ok {
			return collectConds(elsestmt, conds)
		}
		return conds, true
	}
	fn := func(node ast.Node) {
		ifstmt := node.(*ast.IfStmt)
		if seen[ifstmt] {
			// this if-statement is part of an if/else-if chain that we've already processed
			return
		}
		if ifstmt.Else == nil {
			// there can be at most one condition
			return
		}
		conds, ok := collectConds(ifstmt, nil)
		if !ok {
			return
		}
		if len(conds) < 2 {
			return
		}
		counts := map[string]int{}
		for _, cond := range conds {
			s := report.Render(pass, cond)
			counts[s]++
			if counts[s] == 2 {
				report.Nodef(pass, cond, "this condition occurs multiple times in this if/else if chain")
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.IfStmt)(nil)}, fn)
	return nil, nil
}

func CheckSillyBitwiseOps(pass *analysis.Pass) (interface{}, error) {
	// FIXME(dh): what happened here?
	if false {
		for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
			for _, block := range ssafn.Blocks {
				for _, ins := range block.Instrs {
					ins, ok := ins.(*ssa.BinOp)
					if !ok {
						continue
					}

					if c, ok := ins.Y.(*ssa.Const); !ok || c.Value == nil || c.Value.Kind() != constant.Int || c.Uint64() != 0 {
						continue
					}
					switch ins.Op {
					case token.AND, token.OR, token.XOR:
					default:
						// we do not flag shifts because too often, x<<0 is part
						// of a pattern, x<<0, x<<8, x<<16, ...
						continue
					}
					path, _ := astutil.PathEnclosingInterval(code.File(pass, ins), ins.Pos(), ins.Pos())
					if len(path) == 0 {
						continue
					}

					if node, ok := path[0].(*ast.BinaryExpr); !ok || !code.IsIntLiteral(node.Y, "0") {
						continue
					}

					switch ins.Op {
					case token.AND:
						pass.Reportf(ins.Pos(), "x & 0 always equals 0")
					case token.OR, token.XOR:
						pass.Reportf(ins.Pos(), "x %s 0 always equals x", ins.Op)
					}
				}
			}
		}
	}
	fn := func(node ast.Node) {
		binop := node.(*ast.BinaryExpr)
		b, ok := pass.TypesInfo.TypeOf(binop).Underlying().(*types.Basic)
		if !ok {
			return
		}
		if (b.Info() & types.IsInteger) == 0 {
			return
		}
		switch binop.Op {
		case token.AND, token.OR, token.XOR:
		default:
			// we do not flag shifts because too often, x<<0 is part
			// of a pattern, x<<0, x<<8, x<<16, ...
			return
		}
		switch y := binop.Y.(type) {
		case *ast.Ident:
			obj, ok := pass.TypesInfo.ObjectOf(y).(*types.Const)
			if !ok {
				return
			}
			if v, _ := constant.Int64Val(obj.Val()); v != 0 {
				return
			}
			path, _ := astutil.PathEnclosingInterval(code.File(pass, obj), obj.Pos(), obj.Pos())
			if len(path) < 2 {
				return
			}
			spec, ok := path[1].(*ast.ValueSpec)
			if !ok {
				return
			}
			if len(spec.Names) != 1 || len(spec.Values) != 1 {
				// TODO(dh): we could support this
				return
			}
			ident, ok := spec.Values[0].(*ast.Ident)
			if !ok {
				return
			}
			if !isIota(pass.TypesInfo.ObjectOf(ident)) {
				return
			}
			switch binop.Op {
			case token.AND:
				report.Nodef(pass, node,
					"%s always equals 0; %s is defined as iota and has value 0, maybe %s is meant to be 1 << iota?", report.Render(pass, binop), report.Render(pass, binop.Y), report.Render(pass, binop.Y))
			case token.OR, token.XOR:
				report.Nodef(pass, node,
					"%s always equals %s; %s is defined as iota and has value 0, maybe %s is meant to be 1 << iota?", report.Render(pass, binop), report.Render(pass, binop.X), report.Render(pass, binop.Y), report.Render(pass, binop.Y))
			}
		case *ast.BasicLit:
			if !code.IsIntLiteral(binop.Y, "0") {
				return
			}
			switch binop.Op {
			case token.AND:
				report.Nodef(pass, node, "%s always equals 0", report.Render(pass, binop))
			case token.OR, token.XOR:
				report.Nodef(pass, node, "%s always equals %s", report.Render(pass, binop), report.Render(pass, binop.X))
			}
		default:
			return
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func isIota(obj types.Object) bool {
	if obj.Name() != "iota" {
		return false
	}
	c, ok := obj.(*types.Const)
	if !ok {
		return false
	}
	return c.Pkg() == nil
}

func CheckNonOctalFileMode(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		call := node.(*ast.CallExpr)
		sig, ok := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
		if !ok {
			return
		}
		n := sig.Params().Len()
		for i := 0; i < n; i++ {
			typ := sig.Params().At(i).Type()
			if !code.IsType(typ, "os.FileMode") {
				continue
			}

			lit, ok := call.Args[i].(*ast.BasicLit)
			if !ok {
				continue
			}
			if len(lit.Value) == 3 &&
				lit.Value[0] != '0' &&
				lit.Value[0] >= '0' && lit.Value[0] <= '7' &&
				lit.Value[1] >= '0' && lit.Value[1] <= '7' &&
				lit.Value[2] >= '0' && lit.Value[2] <= '7' {

				v, err := strconv.ParseInt(lit.Value, 10, 64)
				if err != nil {
					continue
				}
				report.Node(pass, call.Args[i], fmt.Sprintf("file mode '%s' evaluates to %#o; did you mean '0%s'?", lit.Value, v, lit.Value),
					edit.Fix("fix octal literal", edit.ReplaceWithString(pass.Fset, call.Args[i], "0"+lit.Value)))
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckPureFunctions(pass *analysis.Pass) (interface{}, error) {
	pure := pass.ResultOf[facts.Purity].(facts.PurityResult)

fnLoop:
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		if code.IsInTest(pass, ssafn) {
			params := ssafn.Signature.Params()
			for i := 0; i < params.Len(); i++ {
				param := params.At(i)
				if code.IsType(param.Type(), "*testing.B") {
					// Ignore discarded pure functions in code related
					// to benchmarks. Instead of matching BenchmarkFoo
					// functions, we match any function accepting a
					// *testing.B. Benchmarks sometimes call generic
					// functions for doing the actual work, and
					// checking for the parameter is a lot easier and
					// faster than analyzing call trees.
					continue fnLoop
				}
			}
		}

		for _, b := range ssafn.Blocks {
			for _, ins := range b.Instrs {
				ins, ok := ins.(*ssa.Call)
				if !ok {
					continue
				}
				refs := ins.Referrers()
				if refs == nil || len(code.FilterDebug(*refs)) > 0 {
					continue
				}

				callee := ins.Common().StaticCallee()
				if callee == nil {
					continue
				}
				if callee.Object() == nil {
					// TODO(dh): support anonymous functions
					continue
				}
				if _, ok := pure[callee.Object().(*types.Func)]; ok {
					pass.Reportf(ins.Pos(), "%s is a pure function but its return value is ignored", callee.Name())
				}
			}
		}
	}
	return nil, nil
}

func CheckDeprecated(pass *analysis.Pass) (interface{}, error) {
	deprs := pass.ResultOf[facts.Deprecated].(facts.DeprecatedResult)

	// Selectors can appear outside of function literals, e.g. when
	// declaring package level variables.

	var tfn types.Object
	stack := 0
	fn := func(node ast.Node, push bool) bool {
		if !push {
			stack--
			return false
		}
		stack++
		if stack == 1 {
			tfn = nil
		}
		if fn, ok := node.(*ast.FuncDecl); ok {
			tfn = pass.TypesInfo.ObjectOf(fn.Name)
		}
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		obj := pass.TypesInfo.ObjectOf(sel.Sel)
		if obj.Pkg() == nil {
			return true
		}
		if pass.Pkg == obj.Pkg() || obj.Pkg().Path()+"_test" == pass.Pkg.Path() {
			// Don't flag stuff in our own package
			return true
		}
		if depr, ok := deprs.Objects[obj]; ok {
			// Look for the first available alternative, not the first
			// version something was deprecated in. If a function was
			// deprecated in Go 1.6, an alternative has been available
			// already in 1.0, and we're targeting 1.2, it still
			// makes sense to use the alternative from 1.0, to be
			// future-proof.
			minVersion := deprecated.Stdlib[code.SelectorName(pass, sel)].AlternativeAvailableSince
			if !code.IsGoVersion(pass, minVersion) {
				return true
			}

			if tfn != nil {
				if _, ok := deprs.Objects[tfn]; ok {
					// functions that are deprecated may use deprecated
					// symbols
					return true
				}
			}
			report.Nodef(pass, sel, "%s is deprecated: %s", report.Render(pass, sel), depr.Msg)
			return true
		}
		return true
	}

	fn2 := func(node ast.Node) {
		spec := node.(*ast.ImportSpec)
		var imp *types.Package
		if spec.Name != nil {
			imp = pass.TypesInfo.ObjectOf(spec.Name).(*types.PkgName).Imported()
		} else {
			imp = pass.TypesInfo.Implicits[spec].(*types.PkgName).Imported()
		}

		p := spec.Path.Value
		path := p[1 : len(p)-1]
		if depr, ok := deprs.Packages[imp]; ok {
			report.Nodef(pass, spec, "Package %s is deprecated: %s", path, depr.Msg)
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Nodes(nil, fn)
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.ImportSpec)(nil)}, fn2)
	return nil, nil
}

func callChecker(rules map[string]CallCheck) func(pass *analysis.Pass) (interface{}, error) {
	return func(pass *analysis.Pass) (interface{}, error) {
		return checkCalls(pass, rules)
	}
}

func checkCalls(pass *analysis.Pass, rules map[string]CallCheck) (interface{}, error) {
	ranges := pass.ResultOf[valueRangesAnalyzer].(map[*ssa.Function]vrp.Ranges)
	fn := func(caller *ssa.Function, site ssa.CallInstruction, callee *ssa.Function) {
		obj, ok := callee.Object().(*types.Func)
		if !ok {
			return
		}

		r, ok := rules[lint.FuncName(obj)]
		if !ok {
			return
		}
		var args []*Argument
		ssaargs := site.Common().Args
		if callee.Signature.Recv() != nil {
			ssaargs = ssaargs[1:]
		}
		for _, arg := range ssaargs {
			if iarg, ok := arg.(*ssa.MakeInterface); ok {
				arg = iarg.X
			}
			vr := ranges[site.Parent()][arg]
			args = append(args, &Argument{Value: Value{arg, vr}})
		}
		call := &Call{
			Pass:   pass,
			Instr:  site,
			Args:   args,
			Parent: site.Parent(),
		}
		r(call)
		path, _ := astutil.PathEnclosingInterval(code.File(pass, site), site.Pos(), site.Pos())
		var astcall *ast.CallExpr
		if len(path) > 1 {
			astcall, _ = path[0].(*ast.CallExpr)
		}
		for idx, arg := range call.Args {
			for _, e := range arg.invalids {
				if astcall != nil {
					report.Nodef(pass, astcall.Args[idx], "%s", e)
				} else {
					pass.Reportf(site.Pos(), "%s", e)
				}
			}
		}
		for _, e := range call.invalids {
			pass.Reportf(call.Instr.Common().Pos(), "%s", e)
		}
	}
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		eachCall(ssafn, fn)
	}
	return nil, nil
}

func shortCallName(call *ssa.CallCommon) string {
	if call.IsInvoke() {
		return ""
	}
	switch v := call.Value.(type) {
	case *ssa.Function:
		fn, ok := v.Object().(*types.Func)
		if !ok {
			return ""
		}
		return fn.Name()
	case *ssa.Builtin:
		return v.Name()
	}
	return ""
}

func CheckWriterBufferModified(pass *analysis.Pass) (interface{}, error) {
	// TODO(dh): this might be a good candidate for taint analysis.
	// Taint the argument as MUST_NOT_MODIFY, then propagate that
	// through functions like bytes.Split

	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		sig := ssafn.Signature
		if ssafn.Name() != "Write" || sig.Recv() == nil || sig.Params().Len() != 1 || sig.Results().Len() != 2 {
			continue
		}
		tArg, ok := sig.Params().At(0).Type().(*types.Slice)
		if !ok {
			continue
		}
		if basic, ok := tArg.Elem().(*types.Basic); !ok || basic.Kind() != types.Byte {
			continue
		}
		if basic, ok := sig.Results().At(0).Type().(*types.Basic); !ok || basic.Kind() != types.Int {
			continue
		}
		if named, ok := sig.Results().At(1).Type().(*types.Named); !ok || !code.IsType(named, "error") {
			continue
		}

		for _, block := range ssafn.Blocks {
			for _, ins := range block.Instrs {
				switch ins := ins.(type) {
				case *ssa.Store:
					addr, ok := ins.Addr.(*ssa.IndexAddr)
					if !ok {
						continue
					}
					if addr.X != ssafn.Params[1] {
						continue
					}
					pass.Reportf(ins.Pos(), "io.Writer.Write must not modify the provided buffer, not even temporarily")
				case *ssa.Call:
					if !code.IsCallTo(ins.Common(), "append") {
						continue
					}
					if ins.Common().Args[0] != ssafn.Params[1] {
						continue
					}
					pass.Reportf(ins.Pos(), "io.Writer.Write must not modify the provided buffer, not even temporarily")
				}
			}
		}
	}
	return nil, nil
}

func loopedRegexp(name string) CallCheck {
	return func(call *Call) {
		if len(extractConsts(call.Args[0].Value.Value)) == 0 {
			return
		}
		if !isInLoop(call.Instr.Block()) {
			return
		}
		call.Invalid(fmt.Sprintf("calling %s in a loop has poor performance, consider using regexp.Compile", name))
	}
}

func CheckEmptyBranch(pass *analysis.Pass) (interface{}, error) {
	for _, ssafn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		if ssafn.Syntax() == nil {
			continue
		}
		if code.IsExample(ssafn) {
			continue
		}
		fn := func(node ast.Node) bool {
			ifstmt, ok := node.(*ast.IfStmt)
			if !ok {
				return true
			}
			if ifstmt.Else != nil {
				b, ok := ifstmt.Else.(*ast.BlockStmt)
				if !ok || len(b.List) != 0 {
					return true
				}
				report.PosfFG(pass, ifstmt.Else.Pos(), "empty branch")
			}
			if len(ifstmt.Body.List) != 0 {
				return true
			}
			report.PosfFG(pass, ifstmt.Pos(), "empty branch")
			return true
		}
		Inspect(ssafn.Syntax(), fn)
	}
	return nil, nil
}

func CheckMapBytesKey(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, b := range fn.Blocks {
		insLoop:
			for _, ins := range b.Instrs {
				// find []byte -> string conversions
				conv, ok := ins.(*ssa.Convert)
				if !ok || conv.Type() != types.Universe.Lookup("string").Type() {
					continue
				}
				if s, ok := conv.X.Type().(*types.Slice); !ok || s.Elem() != types.Universe.Lookup("byte").Type() {
					continue
				}
				refs := conv.Referrers()
				// need at least two (DebugRef) references: the
				// conversion and the *ast.Ident
				if refs == nil || len(*refs) < 2 {
					continue
				}
				ident := false
				// skip first reference, that's the conversion itself
				for _, ref := range (*refs)[1:] {
					switch ref := ref.(type) {
					case *ssa.DebugRef:
						if _, ok := ref.Expr.(*ast.Ident); !ok {
							// the string seems to be used somewhere
							// unexpected; the default branch should
							// catch this already, but be safe
							continue insLoop
						} else {
							ident = true
						}
					case *ssa.MapLookup:
					default:
						// the string is used somewhere else than a
						// map lookup
						continue insLoop
					}
				}

				// the result of the conversion wasn't assigned to an
				// identifier
				if !ident {
					continue
				}
				pass.Reportf(conv.Pos(), "m[string(key)] would be more efficient than k := string(key); m[k]")
			}
		}
	}
	return nil, nil
}

func CheckRangeStringRunes(pass *analysis.Pass) (interface{}, error) {
	return sharedcheck.CheckRangeStringRunes(pass)
}

func CheckSelfAssignment(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		assign := node.(*ast.AssignStmt)
		if assign.Tok != token.ASSIGN || len(assign.Lhs) != len(assign.Rhs) {
			return
		}
		for i, lhs := range assign.Lhs {
			rhs := assign.Rhs[i]
			if reflect.TypeOf(lhs) != reflect.TypeOf(rhs) {
				continue
			}
			rlh := report.Render(pass, lhs)
			rrh := report.Render(pass, rhs)
			if rlh == rrh {
				report.PosfFG(pass, assign.Pos(), "self-assignment of %s to %s", rrh, rlh)
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.AssignStmt)(nil)}, fn)
	return nil, nil
}

func buildTagsIdentical(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	s1s := make([]string, len(s1))
	copy(s1s, s1)
	sort.Strings(s1s)
	s2s := make([]string, len(s2))
	copy(s2s, s2)
	sort.Strings(s2s)
	for i, s := range s1s {
		if s != s2s[i] {
			return false
		}
	}
	return true
}

func CheckDuplicateBuildConstraints(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		constraints := buildTags(f)
		for i, constraint1 := range constraints {
			for j, constraint2 := range constraints {
				if i >= j {
					continue
				}
				if buildTagsIdentical(constraint1, constraint2) {
					report.PosfFG(pass, f.Pos(), "identical build constraints %q and %q",
						strings.Join(constraint1, " "),
						strings.Join(constraint2, " "))
				}
			}
		}
	}
	return nil, nil
}

func CheckSillyRegexp(pass *analysis.Pass) (interface{}, error) {
	// We could use the rule checking engine for this, but the
	// arguments aren't really invalid.
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, b := range fn.Blocks {
			for _, ins := range b.Instrs {
				call, ok := ins.(*ssa.Call)
				if !ok {
					continue
				}
				if !code.IsCallToAny(call.Common(), "regexp.MustCompile", "regexp.Compile", "regexp.Match", "regexp.MatchReader", "regexp.MatchString") {
					continue
				}
				c, ok := call.Common().Args[0].(*ssa.Const)
				if !ok {
					continue
				}
				s := constant.StringVal(c.Value)
				re, err := syntax.Parse(s, 0)
				if err != nil {
					continue
				}
				if re.Op != syntax.OpLiteral && re.Op != syntax.OpEmptyMatch {
					continue
				}
				pass.Reportf(call.Pos(), "regular expression does not contain any meta characters")
			}
		}
	}
	return nil, nil
}

func CheckMissingEnumTypesInDeclaration(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		decl := node.(*ast.GenDecl)
		if !decl.Lparen.IsValid() {
			return
		}
		if decl.Tok != token.CONST {
			return
		}

		groups := code.GroupSpecs(pass.Fset, decl.Specs)
	groupLoop:
		for _, group := range groups {
			if len(group) < 2 {
				continue
			}
			if group[0].(*ast.ValueSpec).Type == nil {
				// first constant doesn't have a type
				continue groupLoop
			}
			for i, spec := range group {
				spec := spec.(*ast.ValueSpec)
				if len(spec.Names) != 1 || len(spec.Values) != 1 {
					continue groupLoop
				}
				switch v := spec.Values[0].(type) {
				case *ast.BasicLit:
				case *ast.UnaryExpr:
					if _, ok := v.X.(*ast.BasicLit); !ok {
						continue groupLoop
					}
				default:
					// if it's not a literal it might be typed, such as
					// time.Microsecond = 1000 * Nanosecond
					continue groupLoop
				}
				if i == 0 {
					continue
				}
				if spec.Type != nil {
					continue groupLoop
				}
			}
			var edits []analysis.TextEdit
			typ := group[0].(*ast.ValueSpec).Type
			for _, spec := range group[1:] {
				nspec := *spec.(*ast.ValueSpec)
				nspec.Type = typ
				edits = append(edits, edit.ReplaceWithNode(pass.Fset, spec, &nspec))
			}
			report.Node(pass, group[0], "only the first constant in this group has an explicit type",
				edit.Fix("add type to all constants in group", edits...))
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.GenDecl)(nil)}, fn)
	return nil, nil
}

func CheckTimerResetReturnValue(pass *analysis.Pass) (interface{}, error) {
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, block := range fn.Blocks {
			for _, ins := range block.Instrs {
				call, ok := ins.(*ssa.Call)
				if !ok {
					continue
				}
				if !code.IsCallTo(call.Common(), "(*time.Timer).Reset") {
					continue
				}
				refs := call.Referrers()
				if refs == nil {
					continue
				}
				for _, ref := range code.FilterDebug(*refs) {
					ifstmt, ok := ref.(*ssa.If)
					if !ok {
						continue
					}

					found := false
					for _, succ := range ifstmt.Block().Succs {
						if len(succ.Preds) != 1 {
							// Merge point, not a branch in the
							// syntactical sense.

							// FIXME(dh): this is broken for if
							// statements a la "if x || y"
							continue
						}
						ssautil.Walk(succ, func(b *ssa.BasicBlock) bool {
							if !succ.Dominates(b) {
								// We've reached the end of the branch
								return false
							}
							for _, ins := range b.Instrs {
								// TODO(dh): we should check that
								// we're receiving from the channel of
								// a time.Timer to further reduce
								// false positives. Not a key
								// priority, considering the rarity of
								// Reset and the tiny likeliness of a
								// false positive
								if ins, ok := ins.(*ssa.Recv); ok && code.IsType(ins.Chan.Type(), "<-chan time.Time") {
									found = true
									return false
								}
							}
							return true
						})
					}

					if found {
						pass.Reportf(call.Pos(), "it is not possible to use Reset's return value correctly, as there is a race condition between draining the channel and the new timer expiring")
					}
				}
			}
		}
	}
	return nil, nil
}

var (
	checkToLowerToUpperComparisonQ = pattern.MustParse(`
	(BinaryExpr
		(CallExpr fun@(Function (Or "strings.ToLower" "strings.ToUpper")) [a])
 		tok@(Or "==" "!=")
 		(CallExpr fun [b]))`)
	checkToLowerToUpperComparisonR = pattern.MustParse(`(CallExpr (SelectorExpr (Ident "strings") (Ident "EqualFold")) [a b])`)
)

func CheckToLowerToUpperComparison(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		m, ok := Match(pass, checkToLowerToUpperComparisonQ, node)
		if !ok {
			return
		}
		rn := pattern.NodeToAST(checkToLowerToUpperComparisonR.Root, m.State).(ast.Expr)
		if m.State["tok"].(token.Token) == token.NEQ {
			rn = &ast.UnaryExpr{
				Op: token.NOT,
				X:  rn,
			}
		}

		report.Node(pass, node, "should use strings.EqualFold instead",
			edit.Fix("replace with strings.EqualFold", edit.ReplaceWithNode(pass.Fset, node, rn)))
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.BinaryExpr)(nil)}, fn)
	return nil, nil
}

func CheckUnreachableTypeCases(pass *analysis.Pass) (interface{}, error) {
	// Check if T subsumes V in a type switch. T subsumes V if T is an interface and T's method set is a subset of V's method set.
	subsumes := func(T, V types.Type) bool {
		tIface, ok := T.Underlying().(*types.Interface)
		if !ok {
			return false
		}

		return types.Implements(V, tIface)
	}

	subsumesAny := func(Ts, Vs []types.Type) (types.Type, types.Type, bool) {
		for _, T := range Ts {
			for _, V := range Vs {
				if subsumes(T, V) {
					return T, V, true
				}
			}
		}

		return nil, nil, false
	}

	fn := func(node ast.Node) {
		tsStmt := node.(*ast.TypeSwitchStmt)

		type ccAndTypes struct {
			cc    *ast.CaseClause
			types []types.Type
		}

		// All asserted types in the order of case clauses.
		ccs := make([]ccAndTypes, 0, len(tsStmt.Body.List))
		for _, stmt := range tsStmt.Body.List {
			cc, _ := stmt.(*ast.CaseClause)

			// Exclude the 'default' case.
			if len(cc.List) == 0 {
				continue
			}

			Ts := make([]types.Type, len(cc.List))
			for i, expr := range cc.List {
				Ts[i] = pass.TypesInfo.TypeOf(expr)
			}

			ccs = append(ccs, ccAndTypes{cc: cc, types: Ts})
		}

		if len(ccs) <= 1 {
			// Zero or one case clauses, nothing to check.
			return
		}

		// Check if case clauses following cc have types that are subsumed by cc.
		for i, cc := range ccs[:len(ccs)-1] {
			for _, next := range ccs[i+1:] {
				if T, V, yes := subsumesAny(cc.types, next.types); yes {
					report.Nodef(pass, next.cc, "unreachable case clause: %s will always match before %s", T.String(), V.String())
				}
			}
		}
	}

	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.TypeSwitchStmt)(nil)}, fn)
	return nil, nil
}

var checkSingleArgAppendQ = pattern.MustParse(`(CallExpr (Builtin "append") [_])`)

func CheckSingleArgAppend(pass *analysis.Pass) (interface{}, error) {
	fn := func(node ast.Node) {
		_, ok := Match(pass, checkSingleArgAppendQ, node)
		if !ok {
			return
		}
		report.PosfFG(pass, node.Pos(), "x = append(y) is equivalent to x = y")
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.CallExpr)(nil)}, fn)
	return nil, nil
}

func CheckStructTags(pass *analysis.Pass) (interface{}, error) {
	importsGoFlags := false

	// we use the AST instead of (*types.Package).Imports to work
	// around vendored packages in GOPATH mode. A vendored package's
	// path will include the vendoring subtree as a prefix.
	for _, f := range pass.Files {
		for _, imp := range f.Imports {
			v := imp.Path.Value
			if v[1:len(v)-1] == "github.com/jessevdk/go-flags" {
				importsGoFlags = true
				break
			}
		}
	}

	fn := func(node ast.Node) {
		for _, field := range node.(*ast.StructType).Fields.List {
			if field.Tag == nil {
				continue
			}
			tags, err := parseStructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			if err != nil {
				report.Nodef(pass, field.Tag, "unparseable struct tag: %s", err)
				continue
			}
			for k, v := range tags {
				if len(v) > 1 {
					isGoFlagsTag := importsGoFlags &&
						(k == "choice" || k == "optional-value" || k == "default")
					if !isGoFlagsTag {
						report.Nodef(pass, field.Tag, "duplicate struct tag %q", k)
					}
				}

				switch k {
				case "json":
					checkJSONTag(pass, field, v[0])
				case "xml":
					checkXMLTag(pass, field, v[0])
				}
			}
		}
	}
	pass.ResultOf[inspect.Analyzer].(*inspector.Inspector).Preorder([]ast.Node{(*ast.StructType)(nil)}, fn)
	return nil, nil
}

func checkJSONTag(pass *analysis.Pass, field *ast.Field, tag string) {
	//lint:ignore SA9003 TODO(dh): should we flag empty tags?
	if len(tag) == 0 {
	}
	fields := strings.Split(tag, ",")
	for _, r := range fields[0] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("!#$%&()*+-./:<=>?@[]^_{|}~ ", r) {
			report.Nodef(pass, field.Tag, "invalid JSON field name %q", fields[0])
		}
	}
	var co, cs, ci int
	for _, s := range fields[1:] {
		switch s {
		case "omitempty":
			co++
		case "":
			// allow stuff like "-,"
		case "string":
			cs++
			// only for string, floating point, integer and bool
			T := code.Dereference(pass.TypesInfo.TypeOf(field.Type).Underlying()).Underlying()
			basic, ok := T.(*types.Basic)
			if !ok || (basic.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsString)) == 0 {
				report.Nodef(pass, field.Tag, "the JSON string option only applies to fields of type string, floating point, integer or bool, or pointers to those")
			}
		case "inline":
			ci++
		default:
			report.Nodef(pass, field.Tag, "unknown JSON option %q", s)
		}
	}
	if co > 1 {
		report.Nodef(pass, field.Tag, `duplicate JSON option "omitempty"`)
	}
	if cs > 1 {
		report.Nodef(pass, field.Tag, `duplicate JSON option "string"`)
	}
	if ci > 1 {
		report.Nodef(pass, field.Tag, `duplicate JSON option "inline"`)
	}
}

func checkXMLTag(pass *analysis.Pass, field *ast.Field, tag string) {
	//lint:ignore SA9003 TODO(dh): should we flag empty tags?
	if len(tag) == 0 {
	}
	fields := strings.Split(tag, ",")
	counts := map[string]int{}
	var exclusives []string
	for _, s := range fields[1:] {
		switch s {
		case "attr", "chardata", "cdata", "innerxml", "comment":
			counts[s]++
			if counts[s] == 1 {
				exclusives = append(exclusives, s)
			}
		case "omitempty", "any":
			counts[s]++
		case "":
		default:
			report.Nodef(pass, field.Tag, "unknown XML option %q", s)
		}
	}
	for k, v := range counts {
		if v > 1 {
			report.Nodef(pass, field.Tag, "duplicate XML option %q", k)
		}
	}
	if len(exclusives) > 1 {
		report.Nodef(pass, field.Tag, "XML options %s are mutually exclusive", strings.Join(exclusives, " and "))
	}
}

func CheckImpossibleTypeAssertion(pass *analysis.Pass) (interface{}, error) {
	type entry struct {
		l, r *types.Func
	}

	msc := &pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).Pkg.Prog.MethodSets
	for _, fn := range pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs {
		for _, b := range fn.Blocks {
			for _, instr := range b.Instrs {
				assert, ok := instr.(*ssa.TypeAssert)
				if !ok {
					continue
				}
				var wrong []entry
				left := assert.X.Type()
				right := assert.AssertedType
				righti, ok := right.Underlying().(*types.Interface)

				if !ok {
					// We only care about interface->interface
					// assertions. The Go compiler already catches
					// impossible interface->concrete assertions.
					continue
				}

				ms := msc.MethodSet(left)
				for i := 0; i < righti.NumMethods(); i++ {
					mr := righti.Method(i)
					sel := ms.Lookup(mr.Pkg(), mr.Name())
					if sel == nil {
						continue
					}
					ml := sel.Obj().(*types.Func)
					if types.AssignableTo(ml.Type(), mr.Type()) {
						continue
					}

					wrong = append(wrong, entry{ml, mr})
				}

				if len(wrong) != 0 {
					s := fmt.Sprintf("impossible type assertion; %s and %s contradict each other:",
						types.TypeString(left, types.RelativeTo(pass.Pkg)),
						types.TypeString(right, types.RelativeTo(pass.Pkg)))
					for _, e := range wrong {
						s += fmt.Sprintf("\n\twrong type for %s method", e.l.Name())
						s += fmt.Sprintf("\n\t\thave %s", e.l.Type())
						s += fmt.Sprintf("\n\t\twant %s", e.r.Type())
					}
					pass.Reportf(assert.Pos(), "%s", s)
				}
			}
		}
	}
	return nil, nil
}

func checkWithValueKey(call *Call) {
	arg := call.Args[1]
	T := arg.Value.Value.Type()
	if T, ok := T.(*types.Basic); ok {
		arg.Invalid(
			fmt.Sprintf("should not use built-in type %s as key for value; define your own type to avoid collisions", T))
	}
	if !types.Comparable(T) {
		arg.Invalid(fmt.Sprintf("keys used with context.WithValue must be comparable, but type %s is not comparable", T))
	}
}
