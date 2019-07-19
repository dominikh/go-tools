package pkg

import "fmt"

var sink int

func fn1(x interface{}) {
	_ = func() {
		switch y := x.(type) {
		case int:
			println(y)
		case string:
			println(y)
		default:
			println(y)
		}
	}
}

func fn2() {
	sink = 1
}

func fn3() {
	x := []int{0, 1, 2, 3}
	for i, v := range x {
		fmt.Println(i, v)
		i++
	}

	// FIXME(dh): we get the following CFG, breaking our analysis:
	// 	.0: # entry
	// 		x := []int{0, 1, 2, 3}
	// 		x
	// 		i
	// 		v
	// 		succs: 1
	//
	// 	.1: # range.loop
	// 		succs: 2 3
	//
	// 	.2: # range.body
	// 		fmt.Println(i, v)
	// 		i++
	// 		succs: 1
	//
	// 	.3: # range.done
	// 		return
}

func fn4() {
	x := []int{0, 1, 2, 3}
	for i, v := range x {
		i++
		fmt.Println(i, v)
	}

	// FIXME(dh): we get the following CFG, breaking our analysis:
	// 	.0: # entry
	// 		x := []int{0, 1, 2, 3}
	// 		x
	// 		i
	// 		v
	// 		succs: 1
	//
	// 	.1: # range.loop
	// 		succs: 2 3
	//
	// 	.2: # range.body
	// 		i++
	// 		fmt.Println(i, v)
	// 		succs: 1
	//
	// 	.3: # range.done
	// 		return
}
