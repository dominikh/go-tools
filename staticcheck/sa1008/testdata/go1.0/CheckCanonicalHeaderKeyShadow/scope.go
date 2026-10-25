package pkg

import htp "net/http"

func fn() {
	h := htp.Header{}
	htp := 1
	_ = htp
	const hdr = "foo"
	_ = h[hdr] //@ diag(`keys in http.Header are canonicalized`)
}
