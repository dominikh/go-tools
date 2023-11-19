//go:build go1.19

package pkg

import _ "io/ioutil" //@ diag("has been deprecated")

// We test this in Go 1.19 even though io/ioutil has technically been deprecated since Go 1.16, because only in Go 1.19
// was the proper deprecation marker added.
