// Structlayout prints the memory layout of struct types. Structs may contain a
// combination of fields and padding for memory alignment. Go follows the field
// order as given in the respective code.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"go/types"
	"log"
	"os"

	"honnef.co/go/tools/go/gcsizes"
	"honnef.co/go/tools/lintcmd/version"
	st "honnef.co/go/tools/structlayout"

	"golang.org/x/tools/go/packages"
)

var (
	jsonFlag    = flag.Bool("json", false, "Format data as JSON")
	versionFlag = flag.Bool("version", false, "Print version and exit")
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	if *versionFlag {
		version.Print(version.Version, version.MachineVersion)
		os.Exit(0)
	}

	if len(flag.Args()) != 2 {
		fmt.Fprintln(flag.CommandLine.Output(), "Wrong number of arguments. Need a package and a type.")
		fmt.Fprintln(flag.CommandLine.Output(), "OPTIONS:")
		flag.PrintDefaults()
		os.Exit(1)
	}
	matchPkg := flag.Arg(0)
	matchType := flag.Arg(1)

	// find matching packages
	cfg := &packages.Config{
		Mode:  packages.NeedImports | packages.NeedExportFile | packages.NeedTypes | packages.NeedSyntax,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, matchPkg)
	if err != nil {
		log.Fatal(err)
	}

	// find matching types
	for _, pkg := range pkgs {
		var typ types.Type
		obj := pkg.Types.Scope().Lookup(matchType)
		if obj == nil {
			continue
		}
		typ = obj.Type()

		st, ok := typ.Underlying().(*types.Struct)
		if !ok {
			log.Fatalf("type %s is not a struct", typ)
		}

		fields := sizes(st, types.Unalias(typ).(*types.Named).Obj().Name(), 0, nil)
		if *jsonFlag {
			emitJSON(fields)
		} else {
			emitText(fields)
		}
		return
	}

	log.Fatalf("type %s.%s not found", matchPkg, matchType)
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
	s := gcsizes.ForArch(build.Default.GOARCH)
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
		field.Size = 1
		field.End++
	}
	pad := s.Sizeof(typ) - field.End
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
