package pkg

import (
	"errors"
	fmtalias "fmt"
)

func fn() {
	_ = fmtalias.Errorf("%d", 0)
	_ = errors.New("")
	_ = errors.New(fmtalias.Sprintf("%d", 0)) //@ diag(`should use fmt.Errorf`)
}
