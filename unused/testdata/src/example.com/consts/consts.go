package pkg

const c1 = 1 //@ used(true)

const c2 = 1 //@ used(true)
const c3 = 1 //@ used(true)
const c4 = 1 //@ used(true)
const C5 = 1 //@ used(true)

const (
	c6 = 0 //@ used(true)
	c7     //@ used(true)
	c8     //@ used(true)

	c9  //@ used(false)
	c10 //@ used(false)
	c11 //@ used(false)
)

var _ = []int{c3: 1}

type T1 struct { //@ used(true)
	F1 [c1]int //@ used(true)
}

func init() { //@ used(true)
	_ = []int{c2: 1}
	var _ [c4]int

	_ = c7
}

func Fn() { //@ used(true)
	const X = 1 //@ used(false)
}
