package pkg

import "net/http"

func fn() {
	var r http.Request
	var h http.Header
	var m map[string][]string
	_ = h["foo"] // MATCH /keys in http.Header are canonicalized/
	h["foo"] = nil
	_ = r.Header["foo"] // MATCH /keys in http.Header are canonicalized/
	r.Header["foo"] = nil
	_ = m["foo"]
}
