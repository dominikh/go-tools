package pkg

import (
	"fmt"
	"io"
)

func _(w io.Writer) {
	w.Write([]byte(fmt.Sprint("abc", "de")))   /* want `should use fmt.Fprint` */
	w.Write([]byte(fmt.Sprintf("%T", w)))      /* want `should use fmt.Fprintf` */
	w.Write([]byte(fmt.Sprintln("abc", "de"))) /* want `should use fmt.Fprintln` */
}
