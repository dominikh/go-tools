package pkg

import (
	"crypto/x509"
	"net/http/httputil"
	"path/filepath"
)

func fn() {
	filepath.HasPrefix("", "") //@ diag(`filepath.HasPrefix has been deprecated since Go 1.0 because it shouldn't be used:`)
	_ = httputil.ErrPersistEOF //@ diag(`httputil.ErrPersistEOF has been deprecated since Go 1.0:`)
	_ = httputil.ServerConn{}  //@ diag(`httputil.ServerConn has been deprecated since Go 1.0:`)
	_ = x509.CertificateRequest{}.Attributes
}
