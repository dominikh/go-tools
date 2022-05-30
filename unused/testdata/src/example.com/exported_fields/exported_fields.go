package pkg

type t1 struct { //@ used("t1", true)
	F1 int //@ used("F1", true)
}

type T2 struct { //@ used("T2", true)
	F2 int //@ used("F2", true)
}

var v struct { //@ used("v", true)
	T3 //@ used("T3", true)
}

type T3 struct{} //@ used("T3", true)

func (T3) Foo() {} //@ used("Foo", true)

func init() { //@ used("init", true)
	v.Foo()
}

func init() { //@ used("init", true)
	_ = t1{}
}

type codeResponse struct { //@ used("codeResponse", true)
	Tree *codeNode `json:"tree"` //@ used("Tree", true)
}

type codeNode struct { //@ used("codeNode", true)
}

func init() { //@ used("init", true)
	_ = codeResponse{}
}
