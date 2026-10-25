// Package pkg ...
package pkg

import httpalias "net/http"

func fn() {
	// Check all the supported functions
	httpalias.Error(nil, "", 506)         //@ diag(`http.StatusVariantAlsoNegotiates`)
	httpalias.Redirect(nil, nil, "", 506) //@ diag(`http.StatusVariantAlsoNegotiates`)
	httpalias.StatusText(506)             //@ diag(`http.StatusVariantAlsoNegotiates`)
	httpalias.RedirectHandler("", 506)    //@ diag(`http.StatusVariantAlsoNegotiates`)

	// Don't flag literals with no known constant
	httpalias.StatusText(600)

	// Don't flag constants
	httpalias.StatusText(httpalias.StatusAccepted)

	// Don't flag items on the whitelist (well known codes)
	httpalias.StatusText(404)

	httpalias.Error(fn2())
}

func fn2() (httpalias.ResponseWriter, string, int) { return nil, "", 0 }
