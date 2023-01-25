package pkg

//lint:ignore U1000 consider yourself used
type t1 struct{} //@ used(true)
type t2 struct{} //@ used(true)
type t3 struct{} //@ used(true)

func (t1) fn1() {} //@ used(true)
func (t1) fn2() {} //@ used(true)
func (t1) fn3() {} //@ used(true)

//lint:ignore U1000 be gone
func (t2) fn1() {} //@ used(true)
func (t2) fn2() {} //@ used(false)
func (t2) fn3() {} //@ used(false)

func (t3) fn1() {} //@ used(false)
func (t3) fn2() {} //@ used(false)
func (t3) fn3() {} //@ used(false)

//lint:ignore U1000 consider yourself used
func fn() { //@ used(true)
	var _ t2
	var _ t3
}

//lint:ignore U1000 bye
type t4 struct { //@ used(true)
	x int //@ used(true)
}

func (t4) bar() {} //@ used(true)

//lint:ignore U1000 consider yourself used
type t5 map[int]struct { //@ used(true)
	y int //@ used(true)
}

//lint:ignore U1000 consider yourself used
type t6 interface { //@ used(true)
	foo() //@ used(true)
}

//lint:ignore U1000 consider yourself used
type t7 = struct { //@ used(true)
	z int //@ used(true)
}

//lint:ignore U1000 consider yourself used
type t8 struct{} //@ used(true)

func (t8) fn() { //@ used(true)
	otherFn()
}

func otherFn() {} //@ used(true)
