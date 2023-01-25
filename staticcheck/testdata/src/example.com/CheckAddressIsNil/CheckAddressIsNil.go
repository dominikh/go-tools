package pkg

func fn(x int, y *int) {
	_ = &x == nil //@ diag(`the address of a variable cannot be nil`)
	_ = &y != nil //@ diag(`the address of a variable cannot be nil`)

	if &x != nil { //@ diag(`the address of a variable cannot be nil`)
		println("obviously.")
	}

	if y == nil {
	}
}
