package pkg

import (
	"bytes"
)

type AliasByte = byte
type AliasSlice = []byte
type AliasSlice2 = []AliasByte

func fn() {
	buf := bytes.NewBufferString("str")
	m := map[string]*bytes.Buffer{"key": buf}

	_ = []AliasByte(m["key"].String()) //@ diag(`should use m["key"].Bytes() instead of []AliasByte(m["key"].String())`)
	_ = AliasSlice(m["key"].String())  //@ diag(`should use m["key"].Bytes() instead of AliasSlice(m["key"].String())`)
	_ = AliasSlice2(m["key"].String()) //@ diag(`should use m["key"].Bytes() instead of AliasSlice2(m["key"].String())`)
}
