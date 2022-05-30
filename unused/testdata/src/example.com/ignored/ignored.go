package pkg

//lint:ignore U1000 consider yourself used
type t1 struct{} //@ used("t1", true)
type t2 struct{} //@ used("t2", true)
type t3 struct{} //@ used("t3", true)

func (t1) fn1() {} //@ used("fn1", true)
func (t1) fn2() {} //@ used("fn2", true)
func (t1) fn3() {} //@ used("fn3", true)

//lint:ignore U1000 be gone
func (t2) fn1() {} //@ used("fn1", true)
func (t2) fn2() {} //@ used("fn2", false)
func (t2) fn3() {} //@ used("fn3", false)

func (t3) fn1() {} //@ used("fn1", false)
func (t3) fn2() {} //@ used("fn2", false)
func (t3) fn3() {} //@ used("fn3", false)

//lint:ignore U1000 consider yourself used
func fn() { //@ used("fn", true)
	var _ t2 //@ used("_", true)
	var _ t3 //@ used("_", true)
}

//lint:ignore U1000 bye
type t4 struct { //@ used("t4", true)
	x int //@ used("x", true)
}

func (t4) bar() {} //@ used("bar", true)

//lint:ignore U1000 consider yourself used
type t5 map[int]struct { //@ used("t5", true)
	y int //@ used("y", true)
}

//lint:ignore U1000 consider yourself used
type t6 interface { //@ used("t6", true)
	foo() //@ used("foo", true)
}

//lint:ignore U1000 consider yourself used
type t7 = struct { //@ used("t7", true)
	z int //@ used("z", true)
}

//lint:ignore U1000 consider yourself used
type t8 struct{} //@ used("t8", true)

func (t8) fn() { //@ used("fn", true)
	otherFn()
}

func otherFn() {} //@ used("otherFn", true)
