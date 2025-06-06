// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fakexml

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"honnef.co/go/tools/go/types/typeutil"
	"honnef.co/go/tools/staticcheck/fakereflect"
)

// typeInfo holds details for the xml representation of a type.
type typeInfo struct {
	xmlname *fieldInfo
	fields  []fieldInfo
}

// fieldInfo holds details for the xml representation of a single field.
type fieldInfo struct {
	idx     []int
	name    string
	xmlns   string
	flags   fieldFlags
	parents []string
}

type fieldFlags int

const (
	fElement fieldFlags = 1 << iota
	fAttr
	fCDATA
	fCharData
	fInnerXML
	fComment
	fAny

	fOmitEmpty

	fMode = fElement | fAttr | fCDATA | fCharData | fInnerXML | fComment | fAny

	xmlName = "XMLName"
)

func (f fieldFlags) String() string {
	switch f {
	case fAttr:
		return "attr"
	case fCDATA:
		return "cdata"
	case fCharData:
		return "chardata"
	case fInnerXML:
		return "innerxml"
	case fComment:
		return "comment"
	case fAny:
		return "any"
	case fOmitEmpty:
		return "omitempty"
	case fAny | fAttr:
		return "any,attr"
	default:
		return strconv.Itoa(int(f))
	}
}

var tinfoMap sync.Map // map[reflect.Type]*typeInfo

// getTypeInfo returns the typeInfo structure with details necessary
// for marshaling and unmarshaling typ.
func getTypeInfo(typ fakereflect.TypeAndCanAddr) (*typeInfo, error) {
	if ti, ok := tinfoMap.Load(typ); ok {
		return ti.(*typeInfo), nil
	}

	tinfo := &typeInfo{}
	if typ.IsStruct() && !typeutil.IsTypeWithName(typ.Type, "encoding/xml.Name") {
		n := typ.NumField()
		for i := range n {
			f := typ.Field(i)
			if (!f.IsExported() && !f.Anonymous) || f.Tag.Get("xml") == "-" {
				continue // Private field
			}

			// For embedded structs, embed its fields.
			if f.Anonymous {
				t := f.Type
				if t.IsPtr() {
					t = t.Elem()
				}
				if t.IsStruct() {
					inner, err := getTypeInfo(t)
					if err != nil {
						return nil, err
					}
					if tinfo.xmlname == nil {
						tinfo.xmlname = inner.xmlname
					}
					for _, finfo := range inner.fields {
						finfo.idx = append([]int{i}, finfo.idx...)
						if err := addFieldInfo(typ, tinfo, &finfo); err != nil {
							return nil, err
						}
					}
					continue
				}
			}

			finfo, err := StructFieldInfo(f)
			if err != nil {
				return nil, err
			}

			if f.Name == xmlName {
				tinfo.xmlname = finfo
				continue
			}

			// Add the field if it doesn't conflict with other fields.
			if err := addFieldInfo(typ, tinfo, finfo); err != nil {
				return nil, err
			}
		}
	}

	ti, _ := tinfoMap.LoadOrStore(typ, tinfo)
	return ti.(*typeInfo), nil
}

// StructFieldInfo builds and returns a fieldInfo for f.
func StructFieldInfo(f fakereflect.StructField) (*fieldInfo, error) {
	finfo := &fieldInfo{idx: f.Index}

	// Split the tag from the xml namespace if necessary.
	tag := f.Tag.Get("xml")
	if i := strings.Index(tag, " "); i >= 0 {
		finfo.xmlns, tag = tag[:i], tag[i+1:]
	}

	// Parse flags.
	tokens := strings.Split(tag, ",")
	if len(tokens) == 1 {
		finfo.flags = fElement
	} else {
		tag = tokens[0]
		for _, flag := range tokens[1:] {
			switch flag {
			case "attr":
				finfo.flags |= fAttr
			case "cdata":
				finfo.flags |= fCDATA
			case "chardata":
				finfo.flags |= fCharData
			case "innerxml":
				finfo.flags |= fInnerXML
			case "comment":
				finfo.flags |= fComment
			case "any":
				finfo.flags |= fAny
			case "omitempty":
				finfo.flags |= fOmitEmpty
			}
		}

		// Validate the flags used.
		switch mode := finfo.flags & fMode; mode {
		case 0:
			finfo.flags |= fElement
		case fAttr, fCDATA, fCharData, fInnerXML, fComment, fAny, fAny | fAttr:
			if f.Name == xmlName {
				return nil, fmt.Errorf("cannot use option %s on XMLName field", mode)
			} else if tag != "" && mode != fAttr {
				return nil, fmt.Errorf("cannot specify name together with option ,%s", mode)
			}
		default:
			// This will also catch multiple modes in a single field.
			return nil, fmt.Errorf("invalid combination of options: %q", f.Tag.Get("xml"))
		}
		if finfo.flags&fMode == fAny {
			finfo.flags |= fElement
		}
		if finfo.flags&fOmitEmpty != 0 && finfo.flags&(fElement|fAttr) == 0 {
			return nil, fmt.Errorf("can only use omitempty on elements and attributes")
		}
	}

	// Use of xmlns without a name is not allowed.
	if finfo.xmlns != "" && tag == "" {
		return nil, fmt.Errorf("namespace without name: %q", f.Tag.Get("xml"))
	}

	if f.Name == xmlName {
		// The XMLName field records the XML element name. Don't
		// process it as usual because its name should default to
		// empty rather than to the field name.
		finfo.name = tag
		return finfo, nil
	}

	if tag == "" {
		// If the name part of the tag is completely empty, get
		// default from XMLName of underlying struct if feasible,
		// or field name otherwise.
		if xmlname := lookupXMLName(f.Type); xmlname != nil {
			finfo.xmlns, finfo.name = xmlname.xmlns, xmlname.name
		} else {
			finfo.name = f.Name
		}
		return finfo, nil
	}

	// Prepare field name and parents.
	parents := strings.Split(tag, ">")
	if parents[0] == "" {
		parents[0] = f.Name
	}
	if parents[len(parents)-1] == "" {
		return nil, fmt.Errorf("trailing '>'")
	}
	finfo.name = parents[len(parents)-1]
	if len(parents) > 1 {
		if (finfo.flags & fElement) == 0 {
			return nil, fmt.Errorf("%s chain not valid with %s flag", tag, strings.Join(tokens[1:], ","))
		}
		finfo.parents = parents[:len(parents)-1]
	}

	// If the field type has an XMLName field, the names must match
	// so that the behavior of both marshaling and unmarshaling
	// is straightforward and unambiguous.
	if finfo.flags&fElement != 0 {
		ftyp := f.Type
		xmlname := lookupXMLName(ftyp)
		if xmlname != nil && xmlname.name != finfo.name {
			return nil, fmt.Errorf("name %q conflicts with name %q in %s.XMLName", finfo.name, xmlname.name, ftyp)
		}
	}
	return finfo, nil
}

// lookupXMLName returns the fieldInfo for typ's XMLName field
// in case it exists and has a valid xml field tag, otherwise
// it returns nil.
func lookupXMLName(typ fakereflect.TypeAndCanAddr) (xmlname *fieldInfo) {
	seen := map[fakereflect.TypeAndCanAddr]struct{}{}
	for typ.IsPtr() {
		typ = typ.Elem()
		if _, ok := seen[typ]; ok {
			// Loop in type graph, e.g. 'type P *P'
			return nil
		}
		seen[typ] = struct{}{}
	}
	if !typ.IsStruct() {
		return nil
	}
	for i, n := 0, typ.NumField(); i < n; i++ {
		f := typ.Field(i)
		if f.Name != xmlName {
			continue
		}
		finfo, err := StructFieldInfo(f)
		if err == nil && finfo.name != "" {
			return finfo
		}
		// Also consider errors as a non-existent field tag
		// and let getTypeInfo itself report the error.
		break
	}
	return nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// addFieldInfo adds finfo to tinfo.fields if there are no
// conflicts, or if conflicts arise from previous fields that were
// obtained from deeper embedded structures than finfo. In the latter
// case, the conflicting entries are dropped.
// A conflict occurs when the path (parent + name) to a field is
// itself a prefix of another path, or when two paths match exactly.
// It is okay for field paths to share a common, shorter prefix.
func addFieldInfo(typ fakereflect.TypeAndCanAddr, tinfo *typeInfo, newf *fieldInfo) error {
	var conflicts []int
Loop:
	// First, figure all conflicts. Most working code will have none.
	for i := range tinfo.fields {
		oldf := &tinfo.fields[i]
		if oldf.flags&fMode != newf.flags&fMode {
			continue
		}
		if oldf.xmlns != "" && newf.xmlns != "" && oldf.xmlns != newf.xmlns {
			continue
		}
		minl := min(len(newf.parents), len(oldf.parents))
		for p := range minl {
			if oldf.parents[p] != newf.parents[p] {
				continue Loop
			}
		}
		if len(oldf.parents) > len(newf.parents) {
			if oldf.parents[len(newf.parents)] == newf.name {
				conflicts = append(conflicts, i)
			}
		} else if len(oldf.parents) < len(newf.parents) {
			if newf.parents[len(oldf.parents)] == oldf.name {
				conflicts = append(conflicts, i)
			}
		} else {
			if newf.name == oldf.name {
				conflicts = append(conflicts, i)
			}
		}
	}
	// Without conflicts, add the new field and return.
	if conflicts == nil {
		tinfo.fields = append(tinfo.fields, *newf)
		return nil
	}

	// If any conflict is shallower, ignore the new field.
	// This matches the Go field resolution on embedding.
	for _, i := range conflicts {
		if len(tinfo.fields[i].idx) < len(newf.idx) {
			return nil
		}
	}

	// Otherwise, if any of them is at the same depth level, it's an error.
	for _, i := range conflicts {
		oldf := &tinfo.fields[i]
		if len(oldf.idx) == len(newf.idx) {
			f1 := typ.FieldByIndex(oldf.idx)
			f2 := typ.FieldByIndex(newf.idx)
			return &TagPathError{typ, f1.Name, f1.Tag.Get("xml"), f2.Name, f2.Tag.Get("xml")}
		}
	}

	// Otherwise, the new field is shallower, and thus takes precedence,
	// so drop the conflicting fields from tinfo and append the new one.
	for c := len(conflicts) - 1; c >= 0; c-- {
		i := conflicts[c]
		copy(tinfo.fields[i:], tinfo.fields[i+1:])
		tinfo.fields = tinfo.fields[:len(tinfo.fields)-1]
	}
	tinfo.fields = append(tinfo.fields, *newf)
	return nil
}

// A TagPathError represents an error in the unmarshaling process
// caused by the use of field tags with conflicting paths.
type TagPathError struct {
	Struct       fakereflect.TypeAndCanAddr
	Field1, Tag1 string
	Field2, Tag2 string
}

func (e *TagPathError) Error() string {
	return fmt.Sprintf("%s field %q with tag %q conflicts with field %q with tag %q", e.Struct, e.Field1, e.Tag1, e.Field2, e.Tag2)
}

// value returns v's field value corresponding to finfo.
// It's equivalent to v.FieldByIndex(finfo.idx), but when passed
// initNilPointers, it initializes and dereferences pointers as necessary.
// When passed dontInitNilPointers and a nil pointer is reached, the function
// returns a zero reflect.Value.
func (finfo *fieldInfo) value(v fakereflect.TypeAndCanAddr) fakereflect.TypeAndCanAddr {
	for i, x := range finfo.idx {
		if i > 0 {
			t := v
			if t.IsPtr() && t.Elem().IsStruct() {
				v = v.Elem()
			}
		}
		v = v.Field(x).Type
	}
	return v
}
