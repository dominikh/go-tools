package pkg

type M map[int]int //@ used(true)

func Fn() { //@ used(true)
	var n M
	_ = []M{n}
}
