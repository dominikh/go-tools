// Package pkg ...
package pkg

import . "fmt" // MATCH "should not use dot imports"

var _ = Println
