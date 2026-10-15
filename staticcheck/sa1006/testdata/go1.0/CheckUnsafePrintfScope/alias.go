package pkg

import (
	err "errors"
	"fmt"
)

var _ = err.New

func alias(s string) {
	_ = fmt.Errorf(s) //@ diag(`should use print-style function`)
}
