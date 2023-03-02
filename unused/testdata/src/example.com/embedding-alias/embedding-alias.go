package pkg

type s1 struct{} //@ used("s1", true)

// Make sure the alias is used, and not just the type it points to.
type a1 = s1 //@ used("a1", true)

type E1 struct { //@ used("E1", true)
	a1 //@ used("a1", true)
}

func F1(e E1) { //@ used("F1", true), used("e", true)
	_ = e.a1
}

// Make sure fields get used correctly when embedded multiple times
type s2 struct { //@ used("s2", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
	c int //@ used("c", false)
}

type a2 = s2 //@ used("a2", true)

type E2 struct { //@ used("E2", true)
	a2 //@ used("a2", true)
}

type E3 struct { //@ used("E3", true)
	a2 //@ used("a2", true)
}

func F2(e E2) { //@ used("F2", true), used("e", true)
	_ = e.a
}

func F3(e E3) { //@ used("F3", true), used("e", true)
	_ = e.b
}

// Make sure embedding aliases to unnamed types works
type a4 = struct { //@ used("a4", true)
	a int //@ used("a", true)
	b int //@ used("b", true)
	c int //@ used("c", false)
}

type E4 struct { //@ used("E4", true)
	a4 //@ used("a4", true)
}

type E5 struct { //@ used("E5", true)
	a4 //@ used("a4", true)
}

func F4(e E4) { //@ used("F4", true), used("e", true)
	_ = e.a
}

func F5(e E5) { //@ used("F5", true), used("e", true)
	_ = e.b
}
