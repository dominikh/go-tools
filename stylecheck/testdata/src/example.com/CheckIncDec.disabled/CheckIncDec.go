// Package pkg ...
package pkg

func fn() {
	var x int
	x--
	x++
	x += 1 //@ diag(`should replace x \+= 1 with x\+\+`)
	x -= 1 //@ diag(`should replace x -= 1 with x--`)
	x /= 1
	x += 2
	x -= 2
}
