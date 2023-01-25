package pkg

type t struct { //@ used(true)
	f int //@ used(true)
}

func fn(v *t) { //@ used(true)
	println(v.f)
}

func init() { //@ used(true)
	var v t
	fn(&v)
}
