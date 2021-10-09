package pkg

import (
	"bytes"
	"fmt"
	"io"
)

func _(w io.Writer) {
	w.Write([]byte(fmt.Sprint("abc", "de")))   // want `should use fmt.Fprint`
	w.Write([]byte(fmt.Sprintf("%T", w)))      // want `should use fmt.Fprintf`
	w.Write([]byte(fmt.Sprintln("abc", "de"))) // want `should use fmt.Fprintln`
}

func fn() {
	buf := new(bytes.Buffer)
	var sw io.StringWriter

	buf.WriteString(fmt.Sprint("abc", "de"))   // want `should use fmt.Fprint`
	buf.WriteString(fmt.Sprintf("%T", 0))      // want `should use fmt.Fprintf`
	buf.WriteString(fmt.Sprintln("abc", "de")) // want `should use fmt.Fprintln`

	// We can't suggest fmt.Fprint here. We don't know if sw implements io.Writer.
	sw.WriteString(fmt.Sprint("abc", "de"))
	sw.WriteString(fmt.Sprintf("%T", 0))
	sw.WriteString(fmt.Sprintln("abc", "de"))
}
