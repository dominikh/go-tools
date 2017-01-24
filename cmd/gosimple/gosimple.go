// gosimple detects code that could be rewritten in a simpler way.
package main // import "honnef.co/go/tools/cmd/gosimple"
import (
	"os"

	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/simple"
)

func main() {
	lintutil.ProcessArgs("gosimple", simple.NewChecker(), os.Args[1:])
}
