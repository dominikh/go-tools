// gosimple detects code that could be rewritten in a simpler way.
package main // import "honnef.co/go/tools/cmd/gosimple"
import (
	"flag"
	"fmt"
	"os"

	"honnef.co/go/tools/lint/lintutil"
	"honnef.co/go/tools/simple"
)

var (
	fGen    bool
	fTags   string
	fIgnore string
	fTests  bool
)

func main() {
	flag.BoolVar(&fGen, "generated", false, "Check generated code")
	flag.StringVar(&fTags, "tags", "", "List of `build tags`")
	flag.StringVar(&fIgnore, "ignore", "", "Space separated list of checks to ignore, in the following format: 'import/path/file.go:Check1,Check2,...' Both the import path and file name sections support globbing, e.g. 'os/exec/*_test.go'")
	flag.BoolVar(&fTests, "tests", true, "Include tests")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] [packages]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	c := simple.NewChecker()
	c.CheckGenerated = fGen

	t := "false"
	if fTests {
		t = "true"
	}
	args := []string{
		"-tags", fTags,
		"-ignore", fIgnore,
		"-tests=" + t,
	}

	args = append(args, flag.Args()...)
	lintutil.ProcessArgs("gosimple", c, args)
}
