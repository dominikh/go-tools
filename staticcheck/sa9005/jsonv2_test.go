package sa9005

import (
	"go/token"
	"go/types"
	"testing"
)

func TestJSONv2MethodSignature(t *testing.T) {
	pkg := types.NewPackage("encoding/json/jsontext", "jsontext")
	encoder := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Encoder", nil), types.NewStruct(nil, nil), nil)
	decoder := types.NewNamed(types.NewTypeName(token.NoPos, pkg, "Decoder", nil), types.NewStruct(nil, nil), nil)
	errType := types.Universe.Lookup("error").Type()
	for _, tt := range []struct {
		name     string
		param    types.Type
		result   types.Type
		variadic bool
		want     bool
	}{
		{"MarshalJSONTo", types.NewPointer(encoder), errType, false, true},
		{"UnmarshalJSONFrom", types.NewPointer(decoder), errType, false, true},
		{"MarshalJSONTo", types.Typ[types.Int], errType, false, false},
		{"MarshalJSONTo", types.NewPointer(decoder), errType, false, false},
		{"UnmarshalJSONFrom", types.NewPointer(encoder), errType, false, false},
		{"MarshalJSONTo", types.NewPointer(encoder), types.Typ[types.Int], false, false},
		{"MarshalJSONTo", types.NewSlice(types.NewPointer(encoder)), errType, true, false},
	} {
		sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", tt.param)), types.NewTuple(types.NewVar(token.NoPos, nil, "", tt.result)), tt.variadic)
		if got := isJSONv2Method(sig, tt.name); got != tt.want {
			t.Errorf("%s %s: got %v want %v", tt.name, sig, got, tt.want)
		}
	}
}
