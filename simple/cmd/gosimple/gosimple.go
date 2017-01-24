// gosimple lints the Go source files named on its command line.
package main // import "honnef.co/go/tools/simple/cmd/gosimple"
import (
	"os"

	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/simple"
)

func main() {
	lintutil.ProcessArgs("gosimple", simple.NewChecker(), os.Args[1:])
}
