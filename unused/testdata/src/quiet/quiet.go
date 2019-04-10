package pkg

type iface interface { // MATCH "type iface is unused"
	foo()
}

type t1 struct{} // MATCH "type t1 is unused"
func (t1) foo()  {}

type t2 struct{}

func (t t2) bar(arg int) (ret int) { return 0 } // MATCH "func t2.bar is unused"

func init() {
	_ = t2{}
}

type t3 struct { // MATCH "type t3 is unused"
	a int
	b int
}
