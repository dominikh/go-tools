package pkg

type t181025 struct{} //@ used("t181025", true)

func (t181025) F() {} //@ used("F", true)

// package-level variable after function declaration used to trigger a
// bug in unused.

var V181025 t181025 //@ used("V181025", true)
