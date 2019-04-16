package pkg

type iface interface { // want `iface`
	foo()
}

type t1 struct{} // want `t1`
func (t1) foo()  {}

type t2 struct{}

func (t t2) bar(arg int) (ret int) { return 0 } // want `bar`

func init() {
	_ = t2{}
}

type t3 struct { // want `t3`
	a int
	b int
}
