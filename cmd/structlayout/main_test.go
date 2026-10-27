package main

import (
	"go/types"
	"testing"
)

func TestDisplayNameForAliasToAnonymousStruct(t *testing.T) {
	pkg := types.NewPackage("p", "p")
	alias := types.NewAlias(
		types.NewTypeName(0, pkg, "Alias", nil),
		types.NewStruct(nil, nil),
	)
	if got, want := displayName("Alias", alias), "Alias"; got != want {
		t.Fatalf("displayName = %q, want %q", got, want)
	}
}
