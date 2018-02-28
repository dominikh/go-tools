// gosimple detects code that could be rewritten in a simpler way.
package main
import (
	"os"

	"github.com/cabify/go-tools/lint/lintutil"
	"github.com/cabify/go-tools/simple"
)

func main() {
	fs := lintutil.FlagSet("gosimple")
	gen := fs.Bool("generated", false, "Check generated code")
	fs.Parse(os.Args[1:])
	c := simple.NewChecker()
	c.CheckGenerated = *gen
	cfg := lintutil.CheckerConfig{
		Checker:     c,
		ExitNonZero: true,
	}
	lintutil.ProcessFlagSet([]lintutil.CheckerConfig{cfg}, fs)
}
