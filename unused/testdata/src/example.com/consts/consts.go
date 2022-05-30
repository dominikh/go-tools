package pkg

const c1 = 1 //@ used("c1", true)

const c2 = 1 //@ used("c2", true)
const c3 = 1 //@ used("c3", true)
const c4 = 1 //@ used("c4", true)
const C5 = 1 //@ used("C5", true)

const (
	c6 = 0 //@ used("c6", true)
	c7     //@ used("c7", true)
	c8     //@ used("c8", true)

	c9  //@ used("c9", false)
	c10 //@ used("c10", false)
	c11 //@ used("c11", false)
)

// constants named _ are used, but are not part of constant groups
const (
	c12 = 0 //@ used("c12", false)
	_       //@ used("_", true)
	c13     //@ used("c13", false)
)

var _ = []int{c3: 1} //@ used("_", true)

type T1 struct { //@ used("T1", true)
	F1 [c1]int //@ used("F1", true)
}

func init() { //@ used("init", true)
	_ = []int{c2: 1}
	var _ [c4]int //@ used("_", true)

	_ = c7
}

func Fn() { //@ used("Fn", true)
	const X = 1 //@ used("X", false)
}
