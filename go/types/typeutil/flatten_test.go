package typeutil

import (
	"go/types"
	"testing"
)

func TestFlattenFieldsRepeatedEmbeddedType(t *testing.T) {
	pkg := types.NewPackage("p", "p")
	newNamed := func(name string, underlying types.Type) *types.Named {
		return types.NewNamed(types.NewTypeName(0, pkg, name, nil), underlying, nil)
	}
	embedded := func(name string, typ types.Type) *types.Var {
		return types.NewField(0, pkg, name, typ, true)
	}

	field := types.NewField(0, pkg, "F", types.Typ[types.Int], false)
	leaf := newNamed("T", types.NewStruct([]*types.Var{field}, nil))
	a := newNamed("A", types.NewStruct([]*types.Var{embedded("T", leaf)}, nil))
	b := newNamed("B", types.NewStruct([]*types.Var{embedded("T", leaf)}, nil))
	root := types.NewStruct([]*types.Var{embedded("A", a), embedded("B", b)}, nil)

	got := FlattenFields(root)
	if len(got) != 2 {
		t.Fatalf("FlattenFields returned %d fields, want 2: %#v", len(got), got)
	}
	if got[0].Var.Name() != "F" || got[1].Var.Name() != "F" {
		t.Fatalf("FlattenFields returned fields %q and %q, want F twice", got[0].Var.Name(), got[1].Var.Name())
	}
	if len(got[0].Path) != 3 || got[0].Path[0] != 0 || got[0].Path[1] != 0 || got[0].Path[2] != 0 ||
		len(got[1].Path) != 3 || got[1].Path[0] != 1 || got[1].Path[1] != 0 || got[1].Path[2] != 0 {
		t.Fatalf("FlattenFields paths = %v and %v, want [0 0 0] and [1 0 0]", got[0].Path, got[1].Path)
	}
}
