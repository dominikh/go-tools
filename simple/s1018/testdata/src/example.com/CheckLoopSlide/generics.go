//go:build go1.18

package pkg

func tpfn[T []int]() {
	var n int
	var bs T
	var offset int

	for i := 0; i < n; i++ { //@ diag(`should use copy() instead of loop for sliding slice elements`)
		bs[i] = bs[offset+i]
	}
}
