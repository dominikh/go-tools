// gosimple lints the Go source files named on its command line.
package main // import "honnef.co/go/simple/cmd/gosimple"
import (
	"os"

	"honnef.co/go/lint/lintutil"
	"honnef.co/go/simple"
)

func main() {
	lintutil.ProcessArgs("gosimple", simple.Funcs, os.Args[1:])
}
