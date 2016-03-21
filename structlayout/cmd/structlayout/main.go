package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"go/types"
	"log"
	"os"

	"golang.org/x/tools/go/loader"
)

var fJSON bool

func init() {
	flag.BoolVar(&fJSON, "json", false, "Format data as JSON")
}

type Field struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Start     int64  `json:"start"`
	End       int64  `json:"end"`
	Size      int64  `json:"size"`
	IsPadding bool   `json:"is_padding"`
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

func emitJSON(fields []Field) {
	if fields == nil {
		fields = []Field{}
	}
	json.NewEncoder(os.Stdout).Encode(fields)
}

func emitText(fields []Field) {
	for _, field := range fields {
		if field.IsPadding {
			fmt.Printf("padding: %d-%d (%d bytes)\n", field.Start, field.End, field.Size)
			continue
		}
		fmt.Printf("%s %s: %d-%d (%d bytes)\n", field.Name, field.Type, field.Start, field.End, field.Size)
	}
}
func sizes(typ *types.Struct, prefix string, base int64, out []Field) []Field {
	wordSize := int64(8)
	maxAlign := int64(8)
	switch build.Default.GOARCH {
	case "386", "arm":
		wordSize, maxAlign = 4, 4
	case "amd64p32":
		wordSize = 4
	}
	s := gcSizes{wordSize, maxAlign}

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
			out = append(out, Field{IsPadding: true, Start: pos, End: pos + padding, Size: padding})
			pos += padding
		}
		size := s.Sizeof(field.Type())
		if typ2, ok := field.Type().Underlying().(*types.Struct); ok && typ2.NumFields() != 0 {
			out = sizes(typ2, prefix+"."+field.Name(), pos, out)
		} else {
			out = append(out, Field{
				Name:  prefix + "." + field.Name(),
				Type:  field.Type().String(),
				Start: offsets[i],
				End:   offsets[i] + size,
				Size:  size,
			})

		}
		pos += size
	}
	return out
}
