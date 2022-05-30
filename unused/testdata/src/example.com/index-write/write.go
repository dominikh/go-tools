package pkg

var x int //@ used("x", true)

func Foo() { //@ used("Foo", true)
	var s []int //@ used("s", true)
	s[x] = 0
}
