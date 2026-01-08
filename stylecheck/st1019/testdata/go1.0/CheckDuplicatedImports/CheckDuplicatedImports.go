// Package pkg ...
package pkg

import (
	"fmt" //@ diag(`package "fmt" is being imported more than once`)
	fmt1 "fmt"
	fmt2 "fmt"

	fine "net/http"

	"os" //@ diag(`package "os" is being imported more than once`)
	os1 "os"

	"C"
	_ "unsafe"

	// This imports the package for its side effects, and then again to use it.
	// If we flagged this, the user would have to remove the _ import. But if
	// later the user stopped using the package directly, they'd be prone to
	// removing the import, losing its side effects.
	"net/http/pprof"
	_ "net/http/pprof"

	_ "strconv"
	strconv1 "strconv" //@ diag(`package "strconv" is being imported more than once`)
	strconv2 "strconv"
)

var _ = fmt.Println
var _ = fmt1.Println
var _ = fmt2.Println
var _ = fine.ListenAndServe
var _ = os.Getenv
var _ = os1.Getenv
var _ = pprof.Cmdline
var _ = strconv1.AppendBool
var _ = strconv2.AppendBool
