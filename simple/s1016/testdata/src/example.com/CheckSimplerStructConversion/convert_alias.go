package pkg

type S1 struct {
	A int
}

type S2 struct {
	A int
}

type Alias = S2

// XXX the diagnostics depend on GODEBUG

func foo() {
	v1 := S1{A: 1}
	v2 := Alias{A: 1}

	_ = Alias{A: v1.A} //@ diag(`should convert v1`)
	_ = S1{A: v2.A}    //@ diag(`should convert v2`)
}
