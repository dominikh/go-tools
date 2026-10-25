package pkg

import (
	"bytes"
	stringspkg "strings"
)

type indexer struct{}

func (indexer) Index(string, string) int { return 0 }

func fn() {
	_ = stringspkg.IndexRune("", 'x') > -1 //@ diag(` stringspkg.ContainsRune`)
	_ = stringspkg.IndexRune("", 'x') >= 0 //@ diag(` stringspkg.ContainsRune`)
	_ = stringspkg.IndexRune("", 'x') > 0
	_ = stringspkg.IndexRune("", 'x') >= -1
	_ = stringspkg.IndexRune("", 'x') != -1 //@ diag(` stringspkg.ContainsRune`)
	_ = stringspkg.IndexRune("", 'x') == -1 //@ diag(`!stringspkg.ContainsRune`)
	_ = stringspkg.IndexRune("", 'x') != 0
	_ = stringspkg.IndexRune("", 'x') < 0 //@ diag(`!stringspkg.ContainsRune`)

	_ = stringspkg.IndexAny("", "") > -1 //@ diag(` stringspkg.ContainsAny`)
	_ = stringspkg.IndexAny("", "") >= 0 //@ diag(` stringspkg.ContainsAny`)
	_ = stringspkg.IndexAny("", "") > 0
	_ = stringspkg.IndexAny("", "") >= -1
	_ = stringspkg.IndexAny("", "") != -1 //@ diag(` stringspkg.ContainsAny`)
	_ = stringspkg.IndexAny("", "") == -1 //@ diag(`!stringspkg.ContainsAny`)
	_ = stringspkg.IndexAny("", "") != 0
	_ = stringspkg.IndexAny("", "") < 0 //@ diag(`!stringspkg.ContainsAny`)

	_ = stringspkg.Index("", "") > -1 //@ diag(` stringspkg.Contains`)
	_ = stringspkg.Index("", "") >= 0 //@ diag(` stringspkg.Contains`)
	_ = stringspkg.Index("", "") > 0
	_ = stringspkg.Index("", "") >= -1
	_ = stringspkg.Index("", "") != -1 //@ diag(` stringspkg.Contains`)
	_ = stringspkg.Index("", "") == -1 //@ diag(`!stringspkg.Contains`)
	_ = stringspkg.Index("", "") != 0
	_ = stringspkg.Index("", "") < 0 //@ diag(`!stringspkg.Contains`)

	_ = bytes.IndexRune(nil, 'x') > -1 //@ diag(` bytes.ContainsRune`)
	_ = bytes.IndexAny(nil, "") > -1   //@ diag(` bytes.ContainsAny`)
	_ = bytes.Index(nil, nil) > -1     //@ diag(` bytes.Contains`)

	strings := indexer{}
	bytes := indexer{}
	_ = strings.Index("", "") > -1
	_ = bytes.Index("", "") > -1
}
