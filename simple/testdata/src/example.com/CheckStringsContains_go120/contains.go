package pkg

import (
	"bytes"
	"strings"
)

func fn() {
	_ = bytes.IndexFunc(nil, func(r rune) bool { return false }) != -1
	_ = strings.IndexFunc("", func(r rune) bool { return false }) != -1
}
