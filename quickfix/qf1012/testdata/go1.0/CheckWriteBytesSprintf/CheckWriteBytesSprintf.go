package pkg

import (
	"bytes"
	format "fmt"
	"io"
)

type NotAWriter struct{}

func (NotAWriter) Write(b []byte) {}

func fn1() {
	var w io.Writer
	var w2 NotAWriter

	w.Write([]byte(format.Sprint("abc", "de")))   //@ diag(`Use fmt.Fprint`)
	w.Write([]byte(format.Sprintf("%T", w)))      //@ diag(`Use fmt.Fprintf`)
	w.Write([]byte(format.Sprintln("abc", "de"))) //@ diag(`Use fmt.Fprintln`)

	w2.Write([]byte(format.Sprint("abc", "de")))
}

func fn2() {
	buf := new(bytes.Buffer)
	var sw io.StringWriter

	buf.WriteString(format.Sprint("abc", "de"))   //@ diag(`Use fmt.Fprint`)
	buf.WriteString(format.Sprintf("%T", 0))      //@ diag(`Use fmt.Fprintf`)
	buf.WriteString(format.Sprintln("abc", "de")) //@ diag(`Use fmt.Fprintln`)

	// We can't suggest fmt.Fprint here. We don't know if sw implements io.Writer.
	sw.WriteString(format.Sprint("abc", "de"))
	sw.WriteString(format.Sprintf("%T", 0))
	sw.WriteString(format.Sprintln("abc", "de"))
}

func fn3() {
	var buf bytes.Buffer
	buf.WriteString(format.Sprint("abc", "de")) //@ diag(`Use fmt.Fprint`)
}
