package pkg

func fn() {
	const c = 0
	var x, y int
	type s []int
	_ = make([]int, 1)
	_ = make([]int, c)       // constant of 0 maybe due to debugging, math or platform-specific code
	_ = make([]int, 0)       // length is mandatory for slices, don't suggest removal
	_ = make(s, 0)           // length is mandatory for slices, don't suggest removal
	_ = make(chan int, 0)    // MATCH /when length is zero, length can be omitted/
	_ = make(map[int]int, 0) // MATCH /when length is zero, length can be omitted/
	_ = make([]int, 1, 1)    // MATCH /when length equals capacity, capacity can be omitted/
	_ = make([]int, x, x)    // MATCH /when length equals capacity, capacity can be omitted/
	_ = make([]int, 1, 2)
	_ = make([]int, x, y)
}
