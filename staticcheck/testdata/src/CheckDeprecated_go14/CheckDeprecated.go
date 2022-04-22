package pkg

import (
	"compress/flate"
	"database/sql/driver"
	"net/http"
	"os"
	"syscall"
)

var _ = syscall.StringByteSlice("") //@ diag(`Use ByteSliceFromString instead`)

func fn1(err error) {
	var r *http.Request
	_ = r.Cancel
	_ = syscall.StringByteSlice("") //@ diag(`Use ByteSliceFromString instead`)
	_ = os.SEEK_SET
	var _ flate.ReadError

	var tr *http.Transport
	tr.CancelRequest(nil)

	var conn driver.Conn
	conn.Begin()
}

// Deprecated: Don't use this.
func fn2() {
	_ = syscall.StringByteSlice("")

	anon := func(x int) {
		println(x)
		_ = syscall.StringByteSlice("")
	}
	anon(1)
}
