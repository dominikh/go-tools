// Package pkg ...
package pkg

import _ "fmt"

import _ "fmt"
import _ "fmt"
import _ "fmt"

import _ "fmt"
import "strings"
import _ "fmt"

// This is fine
import _ "fmt"

// This is fine
import _ "fmt"
import _ "fmt"
import _ "fmt"

// This is fine
import _ "fmt"
import "bytes"
import _ "fmt"

import _ "fmt" // This is fine

// This is not fine
import (
	_ "fmt"
)

import (
	_ "fmt"
	"strconv"
	// This is fine
	_ "fmt"
)

var _ = strings.NewReader
var _ = bytes.NewBuffer
var _ = strconv.IntSize

// MATCH:4 "blank import"
// MATCH:6 "blank import"
// MATCH:10 "blank import"
// MATCH:12 "blank import"
// MATCH:25 "blank import"
// MATCH:31 "blank import"
// MATCH:35 "blank import"
