package pkg

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

func fn1() {
	strings.Replace("", "", "", 1) // want `only makes sense if its return values get used`
	foo(1, 2)                      // want `only makes sense if its return values get used`
	baz(1, 2)                      // want `only makes sense if its return values get used`
	_, x := baz(1, 2)
	_ = x
	bar(1, 2)
}

func fn2() {
	r, _ := http.NewRequest("GET", "/", nil)
	r.WithContext(context.Background()) // want `only makes sense if its return values get used`
	http.NewRequest("GET", "/", nil)    // want `only makes sense if its return values get used`
}

func foo(a, b int) int        { return a + b }
func baz(a, b int) (int, int) { return a + b, a + b }
func bar(a, b int) int {
	println(a + b)
	return a + b
}

func empty()            {}
func stubPointer() *int { return nil }
func stubInt() int      { return 0 }

func fn3() {
	empty()
	stubPointer()
	stubInt()
}

func fn4() {
	// We never want to flag any of these
	subtle.ConstantTimeByteEq(0, 0)
	subtle.ConstantTimeCompare(nil, nil)
	subtle.ConstantTimeCopy(0, nil, nil)
	subtle.ConstantTimeEq(0, 0)
	subtle.ConstantTimeLessOrEq(0, 0)
	subtle.ConstantTimeSelect(0, 0, 0)
}

func produceBool(a int) bool {
	return a == 0
}

func fn5() {
	// make sure we do correctly flag pointless calls to produceBool
	produceBool(0) // want `only makes sense if its return values get used`
	if produceBool(0) {
		// only comments, no code. possibly a TODO to actually handle
		// this case. flagging this would be noisy.
	}

	_ = produceBool(0) // assigning to _ counts as a use
	return
	_ = produceBool(0) // this code is dead
}
