package pkg

import "fmt"

func fn1() {
	var x int
	x = gen() //@ diag(`this value of x is never used`)
	x = gen()
	println(x)

	var y int
	if true {
		y = gen() //@ diag(`this value of y is never used`)
	}
	y = gen()
	println(y)
}

func gen() int {
	println() // make it unpure
	return 0
}

func fn2() {
	x, y := gen(), gen() //@ diag(`this value of x is never used`), diag(`this value of y is never used`)
	x, y = gen(), gen()
	println(x, y)
}

func fn3() {
	x := uint32(0)
	if true {
		x = 1
	} else {
		x = 2
	}
	println(x)
}

func gen2() (int, int) {
	println()
	return 0, 0
}

func fn4() {
	x, y := gen2() //@ diag(`this value of x is never used`)
	println(y)
	x, y = gen2() //@ diag(`this value of x is never used`), diag(`this value of y is never used`)
	x, _ = gen2() //@ diag(`this value of x is never used`)
	x, y = gen2()
	println(x, y)
}

func fn5(m map[string]string) {
	v, ok := m[""] //@ diag(`this value of v is never used`), diag(`this value of ok is never used`)
	v, ok = m[""]
	println(v, ok)
}

func fn6() {
	x := gen()
	// Do not report variables if they've been assigned to the blank identifier
	_ = x
}

func fn7() {
	func() {
		var x int
		x = gen() //@ diag(`this value of x is never used`)
		x = gen()
		println(x)
	}()
}

func fn() int { println(); return 0 }

var y = func() {
	v := fn() //@ diag(`never used`)
	v = fn()
	println(v)
}

func fn8() {
	x := gen()
	switch x {
	}

	y := gen() //@ diag(`this value of y is never used`)
	y = gen()
	switch y {
	}

	z, _ := gen2()
	switch z {
	}

	_, a := gen2()
	switch a {
	}

	b, c := gen2() //@ diag(`this value of b is never used`)
	println(c)
	b, c = gen2() //@ diag(`this value of c is never used`)
	switch b {
	}
}

func fn9() {
	xs := []int{}
	for _, x := range xs {
		foo, err := work(x) //@ diag(`this value of foo is never used`)
		if err != nil {
			return
		}
		if !foo {
			continue
		}
	}
}

func work(int) (bool, error) { return false, nil }

func resolveWeakTypes(types []int) {
	for i := range types {
		runEnd := findRunLimit(i)

		if true {
			_ = runEnd
		}
		i = runEnd //@ diag(`this value of i is never used`)
	}
}

func findRunLimit(int) int { return 0 }

func fn10() {
	slice := []string(nil)
	if true {
		slice = []string{"1", "2"}
	} else {
		slice = []string{"3", "4"}
	}
	fmt.Println(slice)
}

func issue1329() {
	{
		n := 1
		n += 1 //@ diag(`this value of n is never used`)
	}
	{
		n := 1
		n ^= 1 //@ diag(`this value of n is never used`)
	}
	{
		n := ""
		n += "" //@ diag(`this value of n is never used`)
	}
	{
		n := 1
		n++ //@ diag(`this value of n is never used`)
	}

	{
		n := 1
		n += 1
		fmt.Println(n)
	}
	{
		n := 1
		n ^= 1
		fmt.Println(n)
	}
	{
		n := ""
		n += ""
		fmt.Println(n)
	}
	{
		n := 1
		n++
		fmt.Println(n)
	}
}
