package pkg

type t struct { //@ used("t", true)
	f int //@ used("f", true)
}

func fn(v *t) { //@ used("fn", true), used("v", true)
	println(v.f)
}

func init() { //@ used("init", true)
	var v t //@ used("v", true)
	fn(&v)
}
