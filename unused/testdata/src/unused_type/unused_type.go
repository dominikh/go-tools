package pkg

type t1 struct{} //@ used(false)

func (t1) Fn() {} //@ used(false)

type t2 struct{} //@ used(true)

func (*t2) Fn() {} //@ used(true)

func init() { //@ used(true)
	(*t2).Fn(nil)
}

type t3 struct{} //@ used(false)

func (t3) fn() //@ used(false)
