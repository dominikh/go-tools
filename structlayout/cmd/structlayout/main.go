package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"go/types"
	"log"
	"os"

	st "honnef.co/go/structlayout"

	"golang.org/x/tools/go/loader"
)

var fJSON bool

func init() {
	flag.BoolVar(&fJSON, "json", false, "Format data as JSON")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	conf := loader.Config{
		Build: &build.Default,
	}

	var pkg string
	var typName string
	pkg = flag.Args()[0]
	typName = flag.Args()[1]
	conf.Import(pkg)

	lprog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}
	var typ types.Type
	obj := lprog.Package(pkg).Pkg.Scope().Lookup(typName)
	if obj == nil {
		log.Fatal("couldn't find type")
	}
	typ = obj.Type()

	st, ok := typ.Underlying().(*types.Struct)
	if !ok {
		log.Fatal("identifier is not a struct type")
	}

	fields := sizes(st, typ.(*types.Named).Obj().Name(), 0, nil)
	if fJSON {
		emitJSON(fields)
	} else {
		emitText(fields)
	}
}

func emitJSON(fields []st.Field) {
	if fields == nil {
		fields = []st.Field{}
	}
	json.NewEncoder(os.Stdout).Encode(fields)
}

func emitText(fields []st.Field) {
	for _, field := range fields {
		fmt.Println(field)
	}
}
func sizes(typ *types.Struct, prefix string, base int64, out []st.Field) []st.Field {
	wordSize := int64(8)
	maxAlign := int64(8)
	switch build.Default.GOARCH {
	case "386", "arm":
		wordSize, maxAlign = 4, 4
	case "amd64p32":
		wordSize = 4
	}
	s := &gcSizes{WordSize: wordSize, MaxAlign: maxAlign}

	n := typ.NumFields()
	var fields []*types.Var
	for i := 0; i < n; i++ {
		fields = append(fields, typ.Field(i))
	}
	offsets := s.Offsetsof(fields)
	for i := range offsets {
		offsets[i] += base
	}

	pos := base
	for i, field := range fields {
		if offsets[i] > pos {
			padding := offsets[i] - pos
			out = append(out, st.Field{
				IsPadding: true,
				Start:     pos,
				End:       pos + padding,
				Size:      padding,
			})
			pos += padding
		}
		size := s.Sizeof(field.Type())
		if typ2, ok := field.Type().Underlying().(*types.Struct); ok && typ2.NumFields() != 0 {
			out = sizes(typ2, prefix+"."+field.Name(), pos, out)
		} else {
			out = append(out, st.Field{
				Name:  prefix + "." + field.Name(),
				Type:  field.Type().String(),
				Start: offsets[i],
				End:   offsets[i] + size,
				Size:  size,
				Align: s.Alignof(field.Type()),
			})
		}
		pos += size
	}

	if len(out) == 0 {
		return out
	}
	field := &out[len(out)-1]
	if field.Size == 0 {
		field := &out[len(out)-1]
		field.Size = 1
		field.End++
	}
	sz := size(out)
	pad := align(sz, s.Alignof(typ)) - sz
	if pad > 0 {
		out = append(out, st.Field{
			IsPadding: true,
			Start:     field.End,
			End:       field.End + pad,
			Size:      pad,
		})

	}

	return out
}

func size(fields []st.Field) int64 {
	n := int64(0)
	for _, field := range fields {
		n += field.Size
	}
	return n
}

// align returns the smallest y >= x such that y % a == 0.
func align(x, a int64) int64 {
	y := x + a - 1
	return y - y%a
}
