// staticcheck detects a myriad of bugs and inefficiencies in your
// code.
package main // import "honnef.co/go/tools/cmd/staticcheck"

import (
	"os"

	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	var args []string
	for _, arg := range os.Args[1:] {
		if arg == "-dubious" {
			continue
		}
		args = append(args, arg)
	}
	c := staticcheck.NewChecker()
	lintutil.ProcessArgs("staticcheck", c, args)
}
