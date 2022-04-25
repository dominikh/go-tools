package pkg

import (
	"io"
	myio "io"
	"os"
)

type T struct{}

func (T) Seek(whence int, offset int64) (int64, error) {
	// This method does NOT implement io.Seeker
	return 0, nil
}

func fn() {
	const SeekStart = 0
	var s io.Seeker
	s.Seek(0, 0)
	s.Seek(0, io.SeekStart)
	s.Seek(io.SeekStart, 0)   //@ diag(`the first argument of io.Seeker is the offset`)
	s.Seek(myio.SeekStart, 0) //@ diag(`the first argument of io.Seeker is the offset`)
	s.Seek(SeekStart, 0)

	var f *os.File
	f.Seek(io.SeekStart, 0) //@ diag(`the first argument of io.Seeker is the offset`)

	var t T
	t.Seek(io.SeekStart, 0) // not flagged, T is not an io.Seeker
}
