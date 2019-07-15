// Package pkg ...
package pkg

import (
	"fmt"      // want `should not import the same package under different names`
	fmt1 "fmt" // want `should not import the same package under different names`
	fmt2 "fmt" // want `should not import the same package under different names`

	fine "net/http"
)

var _ = fmt.Println
var _ = fmt1.Println
var _ = fmt2.Println
var _ = fine.ListenAndServe
