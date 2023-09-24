package pkg

type I1[T any] interface { //@ used("I1", true), used("T", true)
	m1() T //@ used("m1", true)
}

type S1 struct{} //@ used("S1", true)
func (s *S1) m1() string { //@ used("s", true), used("m1", true)
	return ""
}

type I2[T any] interface { //@ used("I2", true), used("T", true)
	m2(T) //@ used("m2", true)
}

type S2 struct{} //@ used("S2", true)
func (s *S2) m2(p string) { //@ used("s", true), used("p", true), used("m2", true)
	return
}

type I3[T any] interface { //@ used("I3", true), used("T", true)
	m3(T) T //@ used("m3", true)
}

type S3_1 struct{} //@ used("S3_1", true)
func (s *S3_1) m3(p string) string { //@ used("s", true), used("p", true), used("m3", true)
	return ""
}

type S3_2 struct{} //@ used("S3_2", true)
func (s *S3_2) m3(p int) string { //@ quiet("s"), quiet("p"), used("m3", false)
	return ""
}

type I4[T, U any] interface { //@ used("I4", true), used("T", true), used("U", true)
	m4_1(T) U //@ used("m4_1", true)
	m4_2() T  //@ used("m4_2", true)
	m4_3() U  //@ used("m4_3", true)
}

type S4_1 struct{} //@ used("S4_1", true)
func (s *S4_1) m4_1(p string) int { //@ used("s", true), used("p", true), used("m4_1", true)
	return 42
}
func (s *S4_1) m4_2() string { //@ used("s", true), used("m4_2", true)
	return ""
}
func (s *S4_1) m4_3() int { //@ used("s", true), used("m4_3", true)
	return 0
}

type S4_2 struct{} //@ used("S4_2", true)
func (s *S4_2) m4_1(p bool) int { //@ quiet("s"), quiet("p"), used("m4_1", false)
	return 0
}
func (s *S4_2) m4_2() string { //@ quiet("s"), used("m4_2", false)
	return ""
}
func (s *S4_2) m4_3() int { //@ quiet("s"), used("m4_3", false)
	return 0
}

type S4_3 struct{} //@ used("S4_3", true)
func (s *S4_3) m4_1(p string) int { //@ quiet("s"), quiet("p"), used("m4_1", false)
	return 42
}
func (s *S4_3) m4_2() int { //@ quiet("s"), used("m4_2", false)
	return 0
}

type I5[T comparable, U comparable] interface { //@ used("I5", true), used("T", true), used("U", true)
	m5(T) U //@ used("m5", true)
}

type S5_1 struct{} //@ used("S5_1", true)
func (s *S5_1) m5(p string) int { //@ used("s", true), used("p", true), used("m5", true)
	return 0
}

type S5_2 struct{} //@ used("S5_2", true)
func (s *S5_2) m5(p any) int { //@ quiet("s"), quiet("p"), used("m5", false)
	return 0
}

type S5_3 struct{} //@ used("S5_3", true)
func (s *S5_3) m5(p string) any { //@ quiet("s"), quiet("p"), used("m5", false)
	return 0
}
