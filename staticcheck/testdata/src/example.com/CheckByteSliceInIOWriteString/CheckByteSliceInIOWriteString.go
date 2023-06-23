package pkg

import (
	"io"
)

func f() {
	var b []byte
	io.WriteString(nil, string(b)) //@ diag(`use io.Writer.Write`)

	type custom []byte
	var c custom
	io.WriteString(nil, string(c)) //@ diag(`use io.Writer.Write`)

	g := func() []byte { return nil }
	io.WriteString(nil, string(g())) //@ diag(`use io.Writer.Write`)

	var d string
	io.WriteString(nil, d)

	io.WriteString(nil, string(123))

	string := func(x []byte) string { return "" }
	io.WriteString(nil, string(b))
}
