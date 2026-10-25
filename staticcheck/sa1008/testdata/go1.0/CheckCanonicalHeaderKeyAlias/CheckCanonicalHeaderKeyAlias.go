package pkg

import httpalias "net/http"

func fn() {
	const hdr = "foo"
	var r httpalias.Request
	h := httpalias.Header{}
	var m map[string][]string
	_ = h["foo"] //@ diag(`keys in http.Header are canonicalized`)
	_ = h[hdr]   //@ diag(`keys in http.Header are canonicalized`)
	h["foo"] = nil
	_ = r.Header["foo"] //@ diag(`keys in http.Header are canonicalized`)
	r.Header["foo"] = nil
	_ = m["foo"]
}
