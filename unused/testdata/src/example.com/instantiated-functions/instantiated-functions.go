package pkg

// https://staticcheck.io/issues/1199

type c1 struct{} //@ used("c1", false)

func Fn[T any]() {} //@ used("Fn", true), used("T", true)

func uncalled() { //@ used("uncalled", false)
	Fn[c1]()
}

type c2 struct{} //@ used("c2", true)

func Called() { //@ used("Called", true)
	Fn[c2]()
}
