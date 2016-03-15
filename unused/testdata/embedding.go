package pkg

type I interface {
	f1()
	f2()
}

type t1 struct{}
type T2 struct{ t1 }

func (t1) f1() {}
func (T2) f2() {}

func Fn() {
	var v T2
	_ = v.t1
}
