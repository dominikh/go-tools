// https://staticcheck.io/issues/1741

package main

import "example.com/issue1741/mid"

func Walk(v mid.Values) bool {
	return v.Values[0].Message() == nil //@ diag(`never true`)
}
