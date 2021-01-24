package pkg

import (
	"io"
	"os"
)

func f() {
	var w = os.Stdout

	b := []byte("abc")
	_, err := io.WriteString(w, string(b)) //@ diag(`use io.Writer.Write`)
	if err != nil {
		panic(err)
	}

	type custom []byte
	c := custom("abc")
	_, err = io.WriteString(w, string(c)) //@ diag(`use io.Writer.Write`)
	if err != nil {
		panic(err)
	}

	g := func() []byte {
		return []byte("abc")
	}
	_, err = io.WriteString(w, string(g())) //@ diag(`use io.Writer.Write`)
	if err != nil {
		panic(err)
	}

	var d string
	_, err = io.WriteString(w, d)
	if err != nil {
		panic(err)
	}

	_, err = io.WriteString(w, string(123))
	if err != nil {
		panic(err)
	}

	h := func() string {
		return "abc"
	}
	_, err = io.WriteString(w, h())
	if err != nil {
		panic(err)
	}

	string := func(x []byte) string {
		return string(x)
	}
	_, err = io.WriteString(w, string(b))
	if err != nil {
		panic(err)
	}
}
