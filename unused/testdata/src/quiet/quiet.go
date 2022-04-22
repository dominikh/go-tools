package pkg

type iface interface { //@ used(false)
	foo()
}

type t1 struct{} //@ used(false)
func (t1) foo()  {} //@ used(false)

type t2 struct{} //@ used(true)

func (t t2) bar(arg int) (ret int) { return 0 } //@ used(false)

func init() { //@ used(true)
	_ = t2{}
}

type t3 struct { //@ used(false)
	a int
	b int
}

type T struct{} //@ used(true)

func fn1() { //@ used(false)
	meh := func(arg T) {
	}
	meh(T{})
}

type localityList []int //@ used(false)

func (l *localityList) Fn1() {} //@ used(false)
func (l *localityList) Fn2() {} //@ used(false)
