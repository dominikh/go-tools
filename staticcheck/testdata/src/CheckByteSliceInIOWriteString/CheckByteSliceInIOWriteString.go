package pkg

import (
	"io"
	"os"
	"strconv"
)

func f() {
	var w = os.Stdout

	b := []byte("abc")
	_, err := io.WriteString(w, string(b)) // want `Use writer Write function instead of WriteString`
	if err != nil {
		panic(err)
	}

	type custom []byte
	c := custom("abc")
	_, err = io.WriteString(w, string(c)) // want `Use writer Write function instead of WriteString`
	if err != nil {
		panic(err)
	}

	g := func() []byte {
		return []byte("abc")
	}
	_, err = io.WriteString(w, string(g())) // want `Use writer Write function instead of WriteString`
	if err != nil {
		panic(err)
	}

	var d string
	_, err = io.WriteString(w, d)
	if err != nil {
		panic(err)
	}

	e := strconv.Itoa(123)
	_, err = io.WriteString(w, e)
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
}
