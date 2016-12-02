package pkg

import "strings"

func fn() {
	println(strings.Trim("\x80test\xff", "\xff")) // MATCH /the second argument to strings.Trim should be a valid UTF-8 encoded string/
	println(strings.Trim("foo", "bar"))
}
