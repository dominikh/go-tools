package pkg

import (
	"bytes"
	str "strings"
)

func fn() {
	str.Replace("", "", "", -1) //@ diag(`could use strings.ReplaceAll instead`)
	str.Replace("", "", "", 0)
	str.Replace("", "", "", 1)

	str.SplitN("", "", -1) //@ diag(`could use strings.Split instead`)
	str.SplitN("", "", 0)
	str.SplitN("", "", 1)

	str.SplitAfterN("", "", -1) //@ diag(`could use strings.SplitAfter instead`)
	str.SplitAfterN("", "", 0)
	str.SplitAfterN("", "", 1)

	bytes.Replace(nil, nil, nil, -1) //@ diag(`could use bytes.ReplaceAll instead`)
	bytes.Replace(nil, nil, nil, 0)
	bytes.Replace(nil, nil, nil, 1)

	bytes.SplitN(nil, nil, -1) //@ diag(`could use bytes.Split instead`)
	bytes.SplitN(nil, nil, 0)
	bytes.SplitN(nil, nil, 1)

	bytes.SplitAfterN(nil, nil, -1) //@ diag(`could use bytes.SplitAfter instead`)
	bytes.SplitAfterN(nil, nil, 0)
	bytes.SplitAfterN(nil, nil, 1)
}
