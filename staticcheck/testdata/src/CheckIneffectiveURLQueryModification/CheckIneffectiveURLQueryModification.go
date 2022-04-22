package pkg

import "net/url"

func fn(u *url.URL) {
	u.Query().Add("", "") //@ diag(`returns a copy`)
	u.Query().Set("", "") //@ diag(`returns a copy`)
	u.Query().Del("")     //@ diag(`returns a copy`)
	u.Query().Encode()

	var t T
	t.Query().Add("", "")
}

type T struct{}

func (v T) Query() T              { return v }
func (v T) Add(arg1, arg2 string) {}
