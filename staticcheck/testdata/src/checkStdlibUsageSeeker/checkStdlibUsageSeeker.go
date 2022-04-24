package pkg

import (
	"io"
	myio "io"
	"os"
)

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
}
