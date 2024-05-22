package pkg

type T1 struct {
	A int
	B int
}

type T2 struct {
	A int
	B int
}

type T3[T any] struct {
	A T
	B T
}

type T4[T any] struct {
	A T
	B T
}

func _() {
	t1 := T1{0, 0}
	t3 := T3[int]{0, 0}

	_ = T2{t1.A, t1.B} //@ diag(`(type T1) to T2`)
	_ = T2{t3.A, t3.B} //@ diag(`(type T3[int]) to T2`)

	_ = T4[int]{t1.A, t1.B} //@ diag(`(type T1) to T4[int]`)
	_ = T4[int]{t3.A, t3.B} //@ diag(`(type T3[int]) to T4[int]`)

	_ = T4[any]{t3.A, t3.B}
}
