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

type T2 struct{}

func (T2) create() (io.ReadCloser, error) {
	return nil, nil
}

func fn3() (T, error) {
	return T{}, nil
}

func fn2() {
	rc, err := fn1()
	defer rc.Close() //@ diag(`should check error returned from fn1() before deferring rc.Close`)
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
	defer t.rc.Close() //@ diag(`should check error returned from fn3() before deferring t.rc.Close`)
	if err != nil {
		println()
	}

	fp, err := os.Open("path")
	defer fp.Close() //@ diag(`should check error returned from os.Open() before deferring fp.Close()`)
	if err != nil {
		println()
	}

	var t2 T2
	rc2, err := t2.create()
	defer rc2.Close() //@ diag(`should check error returned from t2.create() before deferring rc2.Close()`)
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
