package errors

import (
	"errors"
	"io"
	"io/fs"
)

var gErr = errors.New("global")

type myErr struct{}

func (myErr) Error() string { return "" }

func is() {
	err := errors.New("oh noes")

	_ = errors.Is(err, fs.ErrNotExist)
	_ = errors.Is(fs.ErrNotExist, err) //@ diag(`wrong order`)
	if errors.Is(err, fs.ErrNotExist) {
	}
	if errors.Is(fs.ErrNotExist, err) { //@ diag(`wrong order`)
	}

	_ = errors.Is(gErr, fs.ErrNotExist)
	_ = errors.Is(fs.ErrNotExist, gErr) //@ diag(`wrong order`)
	if errors.Is(gErr, fs.ErrNotExist) {
	}
	if errors.Is(fs.ErrNotExist, gErr) { //@ diag(`wrong order`)
	}

	_ = errors.Is(myErr{}, fs.ErrNotExist)
	_ = errors.Is(fs.ErrNotExist, myErr{}) //@ diag(`wrong order`)
	if errors.Is(myErr{}, fs.ErrNotExist) {
	}
	if errors.Is(fs.ErrNotExist, myErr{}) { //@ diag(`wrong order`)
	}

	_ = errors.Is(&myErr{}, fs.ErrNotExist)
	_ = errors.Is(fs.ErrNotExist, &myErr{}) //@ diag(`wrong order`)
	if errors.Is(&myErr{}, fs.ErrNotExist) {
	}
	if errors.Is(fs.ErrNotExist, &myErr{}) { //@ diag(`wrong order`)
	}

	if !errors.Is(io.EOF, fs.ErrNotExist) {
		// If both arguments are globals there's no right or wrong order.
	}
}
