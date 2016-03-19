package pkg

type t1 struct{} // MATCH t1

func (t1) Fn() {}

type t2 struct{}

func (*t2) Fn() {}

func init() {
	(*t2).Fn(nil)
}

type t3 struct{} // MATCH t3

func (t3) fn()
