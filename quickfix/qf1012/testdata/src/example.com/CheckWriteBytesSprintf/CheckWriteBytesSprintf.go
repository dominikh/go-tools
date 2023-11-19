package pkg

import (
	"bytes"
	"fmt"
	"io"
)

type NotAWriter struct{}

func (NotAWriter) Write(b []byte) {}

func fn1() {
	var w io.Writer
	var w2 NotAWriter

	w.Write([]byte(fmt.Sprint("abc", "de")))   //@ diag(`Use fmt.Fprint`)
	w.Write([]byte(fmt.Sprintf("%T", w)))      //@ diag(`Use fmt.Fprintf`)
	w.Write([]byte(fmt.Sprintln("abc", "de"))) //@ diag(`Use fmt.Fprintln`)

	w2.Write([]byte(fmt.Sprint("abc", "de")))
}

func fn2() {
	buf := new(bytes.Buffer)
	var sw io.StringWriter

	buf.WriteString(fmt.Sprint("abc", "de"))   //@ diag(`Use fmt.Fprint`)
	buf.WriteString(fmt.Sprintf("%T", 0))      //@ diag(`Use fmt.Fprintf`)
	buf.WriteString(fmt.Sprintln("abc", "de")) //@ diag(`Use fmt.Fprintln`)

	// We can't suggest fmt.Fprint here. We don't know if sw implements io.Writer.
	sw.WriteString(fmt.Sprint("abc", "de"))
	sw.WriteString(fmt.Sprintf("%T", 0))
	sw.WriteString(fmt.Sprintln("abc", "de"))
}
