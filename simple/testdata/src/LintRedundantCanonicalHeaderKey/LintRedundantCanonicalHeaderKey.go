package pkg

import "net/http"

func fn1() {
	var headers http.Header
	headers.Add(http.CanonicalHeaderKey("test"), "test") // want `calling net/http.CanonicalHeaderKey on the key of`
	headers.Del(http.CanonicalHeaderKey("test"))         // want `calling net/http.CanonicalHeaderKey on the key of`
	headers.Get(http.CanonicalHeaderKey("test"))         // want `calling net/http.CanonicalHeaderKey on the key of`
	headers.Set(http.CanonicalHeaderKey("test"), "test") // want `calling net/http.CanonicalHeaderKey on the key of`

	headers.Add("test", "test")
	headers.Del("test")
	headers.Get("test")
	headers.Set("test", "test")
}
