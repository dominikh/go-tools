package pkg

import (
	"io"
	"os"
)

func f() {
	var w = os.Stdout

	b := []byte("abc")
	_, err := io.WriteString(w, string(b)) // want `Use writer Write function instead of WriteString`
	if err != nil {
		panic(err)
	}
}
