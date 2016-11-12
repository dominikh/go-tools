package pkg

import "strings"

func fn() {
	strings.Trim("\x80test\xff", "\xff") // MATCH /the second argument to strings.Trim should be a valid UTF-8 encoded string/
	strings.Trim("foo", "bar")
}
