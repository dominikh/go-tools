package pkg

import (
	"io"
	"os"
)

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
func (s *S5_2) m5(p any) int { //@ used("s", true), used("p", true), used("m5", true)
	return 0
}

type S5_3 struct{} //@ used("S5_3", true)
func (s *S5_3) m5(p string) any { //@ used("s", true), used("p", true), used("m5", true)
	return 0
}

type S5_4 struct{} //@ used("S5_4", true)
func (s *S5_4) m5(p string) io.Reader { //@ used("s", true), used("p", true), used("m5", true)
	return nil
}

type I6[R io.Reader, W io.Writer] interface { //@ used("I6", true), used("R", true), used("W", true)
	m6_1(R) R //@ used("m6_1", true)
	m6_2(W) W //@ used("m6_2", true)
}

type S6_1 struct{} //@ used("S6_1", true)
func (s *S6_1) m6_1(p io.Reader) io.Reader { //@ used("s", true), used("p", true), used("m6_1", true)
	return p
}
func (s *S6_1) m6_2(p io.Writer) io.Writer { //@ used("s", true), used("p", true), used("m6_2", true)
	return p
}

type S6_2 struct{} //@ used("S6_2", true)
func (s *S6_2) m6_1(p io.ReadCloser) io.ReadCloser { //@ used("s", true), used("p", true), used("m6_1", true)
	return p
}
func (s *S6_2) m6_2(p io.WriteCloser) io.WriteCloser { //@ used("s", true), used("p", true), used("m6_2", true)
	return p
}

type S6_3 struct{} //@ used("S6_3", true)
func (s *S6_3) m6_1(p int) int { //@ quiet("s"), quiet("p"), used("m6_1", false)
	return p
}
func (s *S6_3) m6_2(p io.Writer) io.Writer { //@ quiet("s"), quiet("p"), used("m6_2", false)
	return p
}

type S6_4 struct{} //@ used("S6_4", true)
func (s *S6_4) m6_1(p *os.File) *os.File { //@ used("s", true), used("p", true), used("m6_1", true)
	return p
}
func (s *S6_4) m6_2(p *os.File) *os.File { //@ used("s", true), used("p", true), used("m6_2", true)
	return p
}

type S6_5 struct{} //@ used("S6_5", true)
func (s *S6_5) m6_1(p os.File) os.File { //@ quiet("s"), quiet("p"), used("m6_1", false)
	return p
}
func (s *S6_5) m6_2(p os.File) os.File { //@ quiet("s"), quiet("p"), used("m6_2", false)
	return p
}

type I7[T ~int | ~string] interface { //@ used("I7", true), used("T", true)
	m7() T //@ used("m7", true)
}

type S7_1 struct{} //@ used("S7_1", true)
func (s *S7_1) m7() int { //@ used("s", true), used("m7", true)
	return 0
}

type S7_2 struct{} //@ used("S7_2", true)
func (s *S7_2) m7() string { //@ used("s", true), used("m7", true)
	return ""
}

type S7_3 struct{} //@ used("S7_3", true)
func (s *S7_3) m7() float32 { //@ quiet("s"), used("m7", false)
	return 0
}

type S7_4 struct{} //@ used("S7_4", true)
func (s *S7_4) m7() any { //@ quiet("s"), used("m7", false)
	return nil
}

type I8[T io.Reader] interface { //@ used("I8", true), used("T", true)
	m8() []T //@ used("m8", true)
}

type S8_1 struct{} //@ used("S8_1", true)
// This should be considered as used obviously. It's known incompleteness that we want to improve.
// This test case just verifies that it doesn't crash.
func (s *S8_1) m8() []io.Reader { //@ quiet("s"), used("m8", false)
	return nil
}

type S8 struct{}           //@ used("S8", true)
type I9[T any] interface { //@ used("I9", true), used("T", true)
	make() *T //@ used("make", true)
}

type S9 struct{} //@ used("S9", true)

func (S9) make() *S8 { return nil } //@ used("make", true)

func i9use(i I9[S8]) { i.make() } //@ used("i9use", true), used("i", true)

func init() { //@ used("init", true)
	i9use(S9{})
}
