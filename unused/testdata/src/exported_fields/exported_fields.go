package pkg

type t1 struct { //@ used(true)
	F1 int //@ used(true)
}

type T2 struct { //@ used(true)
	F2 int //@ used(true)
}

var v struct { //@ used(true)
	T3 //@ used(true)
}

type T3 struct{} //@ used(true)

func (T3) Foo() {} //@ used(true)

func init() { //@ used(true)
	v.Foo()
}

func init() { //@ used(true)
	_ = t1{}
}

type codeResponse struct { //@ used(true)
	Tree *codeNode `json:"tree"` //@ used(true)
}

type codeNode struct { //@ used(true)
}

func init() { //@ used(true)
	_ = codeResponse{}
}
