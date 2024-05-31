package pkg

import (
	"io"
	"os"
)

func fn1() (io.ReadCloser, error) {
	return nil, nil
}

type T struct {
	rc io.ReadCloser
}

func fn3() (T, error) {
	return T{}, nil
}

func fn2() {
	rc, err := fn1()
	defer rc.Close() //@ diag(`should check returned error from fn1() before deferring rc.Close`)
	if err != nil {
		println()
	}

	rc, _ = fn1()
	defer rc.Close()

	rc, err = fn1()
	if err != nil {
		println()
	}
	defer rc.Close()

	t, err := fn3()
	defer t.rc.Close() //@ diag(`should check returned error from fn3() before deferring t.rc.Close`)
	if err != nil {
		println()
	}

	fp, err := os.Open("path")
	defer fp.Close() //@ diag(`should check returned error from os.Open() before deferring fp.Close()`)
	if err != nil {
		println()
	}

	// Don't flag this, we're closing a different reader
	x, err := fn1()
	defer rc.Close()
	if err != nil {
		println()
	}
	_ = x
}
