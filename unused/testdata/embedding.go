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

type I2 interface {
	f3()
	f4()
}

var _ I2 = &t4{}

type t3 struct{}
type t4 struct{ t3 }

func (*t3) f3() {}
func (*t4) f4() {}

var _ = t4{}.t3 // XXX
