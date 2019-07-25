// Package pkg ...
package pkg

import (
	"fmt"      // want `should not import the same package multiple times`
	fmt1 "fmt" // want `should not import the same package multiple times`
	fmt2 "fmt" // want `should not import the same package multiple times`

	fine "net/http"

	"os"     // want `should not import the same package multiple times`
	os1 "os" // want `should not import the same package multiple times`

	"C"
	_ "unsafe"
)

var _ = fmt.Println
var _ = fmt1.Println
var _ = fmt2.Println
var _ = fine.ListenAndServe
var _ = os.Getenv
var _ = os1.Getenv
