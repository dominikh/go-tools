package pkg

import (
	"compress/flate"
	"crypto/x509"
	"database/sql/driver"
	"net/http"
	"os"
	"syscall"
)

var _ = syscall.StringByteSlice("") //@ diag(`Use ByteSliceFromString instead`)

func fn1(err error) {
	var r http.Request
	var rp *http.Request
	_ = r.Cancel                        //@ diag(re`deprecated since Go 1\.7:.+If a Request's Cancel field and context are both`)
	_ = rp.Cancel                       //@ diag(re`deprecated since Go 1\.7:.+If a Request's Cancel field and context are both`)
	_ = syscall.StringByteSlice("")     //@ diag(`Use ByteSliceFromString instead`)
	_ = os.SEEK_SET                     //@ diag(`Use io.SeekStart, io.SeekCurrent, and io.SeekEnd`)
	_ = os.SEEK_CUR                     //@ diag(`Use io.SeekStart, io.SeekCurrent, and io.SeekEnd`)
	_ = os.SEEK_END                     //@ diag(`Use io.SeekStart, io.SeekCurrent, and io.SeekEnd`)
	if err == http.ErrWriteAfterFlush { //@ diag(`ErrWriteAfterFlush is no longer`)
		println()
	}
	var _ flate.ReadError //@ diag(`No longer returned`)

	var tr *http.Transport
	tr.CancelRequest(nil) //@ diag(`CancelRequest has been deprecated`)

	var conn driver.Conn
	conn.Begin() //@ diag(`Begin has been deprecated`)

	_ = x509.CertificateRequest{}.Attributes //@ diag(`x509.CertificateRequest{}.Attributes has been deprecated since Go 1.5 and an alternative has been available since Go 1.3:`)
}

// Deprecated: Don't use this.
func fn2() {
	_ = syscall.StringByteSlice("")

	anon := func(x int) {
		println(x)
		_ = syscall.StringByteSlice("")

		anon := func(x int) {
			println(x)
			_ = syscall.StringByteSlice("")
		}
		anon(2)
	}
	anon(1)
}
