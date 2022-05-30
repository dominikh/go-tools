package pkg

type t1 struct{} //@ used("t1", false)

func (t1) Fn() {} //@ used("Fn", false)

type t2 struct{} //@ used("t2", true)

func (*t2) Fn() {} //@ used("Fn", true)

func init() { //@ used("init", true)
	(*t2).Fn(nil)
}

type t3 struct{} //@ used("t3", false)

func (t3) fn() //@ used("fn", false)
