package pkg

import "io"

func fn1() (io.ReadCloser, error) {
	return nil, nil
}

func fn2() {
	rc, err := fn1()
	defer rc.Close() // MATCH /should check returned error before deferring rc.Close/
	if err != nil {
	}

	rc, _ = fn1()
	defer rc.Close()

	rc, err = fn1()
	if err != nil {
	}
	defer rc.Close()
}
