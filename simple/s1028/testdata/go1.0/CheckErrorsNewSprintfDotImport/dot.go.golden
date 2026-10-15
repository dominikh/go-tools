package pkg

import (
	"errors"
	. "fmt"
)

func fn() {
	_ = errors.New(Sprintf("%d", 0)) //@ diag(`should use fmt.Errorf`)
}
