// Package pkg ...
package pkg

type Alias = error

func fn10() (Alias, int) { return nil, 0 } //@ diag(`error should be returned as the last argument`)
