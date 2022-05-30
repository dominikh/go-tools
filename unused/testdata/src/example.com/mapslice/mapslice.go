package pkg

type M map[int]int //@ used("M", true)

func Fn() { //@ used("Fn", true)
	var n M //@ used("n", true)
	_ = []M{n}
}
