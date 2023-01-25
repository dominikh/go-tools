// Package pkg ...
package pkg

import _ "fmt" //@ diag(`blank import`)

import _ "fmt" //@ diag(`blank import`)
import _ "fmt"
import _ "fmt"

import _ "fmt" //@ diag(`blank import`)
import "strings"
import _ "fmt" //@ diag(`blank import`)

// This is fine
import _ "fmt"

// This is fine
import _ "fmt"
import _ "fmt"
import _ "fmt"

// This is fine
import _ "fmt"
import "bytes"
import _ "fmt" //@ diag(`blank import`)

import _ "fmt" // This is fine

// This is not fine
import (
	_ "fmt" //@ diag(`blank import`)
)

import (
	_ "fmt" //@ diag(`blank import`)
	"strconv"
	// This is fine
	_ "fmt"
)

var _ = strings.NewReader
var _ = bytes.NewBuffer
var _ = strconv.IntSize
