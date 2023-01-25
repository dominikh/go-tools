// Package pkg ...
package pkg

import . "fmt" //@ diag(`should not use dot imports`)

var _ = Println
