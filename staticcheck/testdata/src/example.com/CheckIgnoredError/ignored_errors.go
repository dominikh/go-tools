package pkg

import (
	"fmt"
)

func returnsError(n int) error {
	fmt.Printf("dummy side effect\n")
	if n < 0 {
		return fmt.Errorf("n is negative")
	}
	return nil
}

func noError(n int) (error, int) {
	fmt.Printf("dummy side effect\n")
	return nil, n * 2
}

func Test() {
	returnsError(1)     //@ diag(`The error returned from calling 'returnsError' is ignored. Do not ignore returned errors as this might hide bugs.`)
	_ = returnsError(2) // explicitly ignoring the error is okay
	noError(3)          // if the error is not the last returned value, then there is no warning (already covered by ST1008)
}

func InIf() {
	if returnsError(4); true { //@ diag(`The error returned from calling 'returnsError' is ignored. Do not ignore returned errors as this might hide bugs.`)
	}
}
