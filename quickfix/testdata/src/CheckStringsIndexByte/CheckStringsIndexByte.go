package pkg

import (
	"bytes"
	"strings"
)

func fn() {
	strings.Index("", "0")            // want `could use strings.IndexByte instead of strings.Index`
	strings.LastIndex("", "0")        // want `could use strings.LastIndexByte instead of strings.LastIndex`
	strings.IndexByte("", '0')        // want `could use strings.Index instead of strings.IndexByte`
	strings.LastIndexByte("", '0')    // want `could use strings.LastIndex instead of strings.LastIndexByte`
	bytes.Index(nil, []byte{'0'})     // want `could use bytes.IndexByte instead of bytes.Index`
	bytes.LastIndex(nil, []byte{'0'}) // want `could use bytes.LastIndexByte instead of bytes.LastIndex`
	bytes.Index(nil, []byte("0"))     // want `could use bytes.IndexByte instead of bytes.Index`

	strings.Index("", "µ")
	strings.Index("", "00")
	strings.LastIndex("", "00")
	bytes.LastIndex(nil, []byte{'0', '0'})
	bytes.Index(nil, []byte("µ"))
}
