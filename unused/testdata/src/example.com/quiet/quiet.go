package pkg

type iface interface { //@ used("iface", false)
	foo() //@ quiet("foo")
}

type t1 struct{} //@ used("t1", false)
func (t1) foo()  {} //@ used("foo", false)

type t2 struct{} //@ used("t2", true)

func (t t2) bar(arg int) (ret int) { //@ quiet("t"), used("bar", false), quiet("arg"), quiet("ret")
	return 0
}

func init() { //@ used("init", true)
	_ = t2{}
}

type t3 struct { //@ used("t3", false)
	a int //@ quiet("a")
	b int //@ quiet("b")
}

type T struct{} //@ used("T", true)

func fn1() { //@ used("fn1", false)
	meh := func(arg T) { //@ quiet("meh"), quiet("arg")
	}
	meh(T{})
}

type localityList []int //@ used("localityList", false)

func (l *localityList) Fn1() {} //@ quiet("l"), used("Fn1", false)
func (l *localityList) Fn2() {} //@ quiet("l"), used("Fn2", false)
