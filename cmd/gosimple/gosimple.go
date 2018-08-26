// gosimple detects code that could be rewritten in a simpler way.
package main // import "honnef.co/go/tools/cmd/gosimple"
import (
	"fmt"
	"os"

	"honnef.co/go/tools/lint"
	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/simple"
)

func main() {
	fmt.Fprintln(os.Stderr, "Gosimple has been deprecated. Please use staticcheck instead.")
	fs := lintutil.FlagSet("gosimple")
	gen := fs.Bool("generated", false, "Check generated code")
	fs.Parse(os.Args[1:])
	c := simple.NewChecker()
	c.CheckGenerated = *gen
	lintutil.ProcessFlagSet([]lint.Checker{c}, fs)
}
