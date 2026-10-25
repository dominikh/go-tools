package pkg

import (
	errpkg "errors"
	"fmt"
)

var _ = errpkg.New

func shadow(s string) {
	errpkg := 1
	_ = errpkg
	_ = fmt.Errorf(s) //@ diag(`should use print-style function`)
}
