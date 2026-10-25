package pkg

import (
	"bytes"
	. "fmt"
)

func fn() {
	var buf bytes.Buffer
	buf.WriteString(Sprint("abc", "de")) //@ diag(`Use fmt.Fprint`)
}
