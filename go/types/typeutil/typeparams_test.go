//go:build go1.18

package typeutil

import (
	"go/types"
	"testing"
)

func simpleUnionIface(terms ...*types.Term) *types.Interface {
	return types.NewInterfaceType(nil, []types.Type{types.NewUnion(terms)})
}

func newChannelIface(chans ...types.Type) *types.Interface {
	var terms []*types.Term
	for _, ch := range chans {
		terms = append(terms, types.NewTerm(false, ch))
	}
	return simpleUnionIface(terms...)
}

func TestTypeSetCoreType(t *testing.T) {
	pkg := types.NewPackage("pkg", "pkg")
	TInt := types.Universe.Lookup("int").Type()
	TUint := types.Universe.Lookup("uint").Type()
	TMyInt1 := types.NewNamed(types.NewTypeName(0, pkg, "MyInt1", nil), TInt, nil)
	TMyInt2 := types.NewNamed(types.NewTypeName(0, pkg, "MyInt2", nil), TInt, nil)
	TChanInt := types.NewChan(types.SendRecv, TInt)
	TChanIntRecv := types.NewChan(types.RecvOnly, TInt)
	TChanIntSend := types.NewChan(types.SendOnly, TInt)
	TNamedChanInt := types.NewNamed(types.NewTypeName(0, pkg, "NamedChan", nil), TChanInt, nil)

	tt := []struct {
		iface *types.Interface
		want  types.Type
	}{
		// same underlying type
		{
			simpleUnionIface(types.NewTerm(false, TMyInt1), types.NewTerm(false, TMyInt2)),
			types.Universe.Lookup("int").Type(),
		},
		// different underlying types
		{
			simpleUnionIface(types.NewTerm(false, TInt), types.NewTerm(false, TUint)),
			nil,
		},
		// empty type set
		{
			types.NewInterfaceType(nil, []types.Type{
				types.NewUnion([]*types.Term{types.NewTerm(false, TInt)}),
				types.NewUnion([]*types.Term{types.NewTerm(false, TUint)}),
			}),
			nil,
		},
	}
	for _, tc := range tt {
		ts := NewTypeSet(tc.iface)
		if !types.Identical(ts.CoreType(), tc.want) {
			t.Errorf("CoreType(%s) = %s, want %s", tc.iface, ts.CoreType(), tc.want)
		}
	}

	tt2 := []struct {
		iface *types.Interface
		want  types.ChanDir
	}{
		{
			// named sr + unnamed sr = sr
			// sr + sr = sr
			newChannelIface(TNamedChanInt, TChanInt),
			types.SendRecv,
		},
		{
			// sr + sr = sr
			newChannelIface(TChanInt, TChanInt),
			types.SendRecv,
		},
		{
			// s + s = s
			newChannelIface(TChanIntSend, TChanIntSend),
			types.SendOnly,
		},
		{
			// r + r = r
			newChannelIface(TChanIntRecv, TChanIntRecv),
			types.RecvOnly,
		},
		{
			// s + r = nil
			newChannelIface(TChanIntSend, TChanIntRecv),
			-1,
		},
		{
			// sr + r = r
			newChannelIface(TChanInt, TChanIntRecv),
			types.RecvOnly,
		},
		{
			// sr + s = s
			newChannelIface(TChanInt, TChanIntSend),
			types.SendOnly,
		},
	}
	for _, tc := range tt2 {
		ts := NewTypeSet(tc.iface)
		core := ts.CoreType()
		if (core == nil) != (tc.want == -1) {
			t.Errorf("CoreType(%s) = %s, want %d", tc.iface, core, tc.want)
		}
		if core == nil {
			continue
		}
		dir := core.(*types.Chan).Dir()
		if dir != tc.want {
			t.Errorf("direction of %s is %d, want %d", tc.iface, dir, tc.want)
		}
	}
}
