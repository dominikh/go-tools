//go:build go1.18

package pkg

func fn1() {
	_ = make(chan int, 0) //@ diag(`should use make(chan int) instead`)
}

func fn2[T chan int]() {
	_ = make(T, 0) //@ diag(`should use make(T) instead`)
}

func fn3[T chan T]() {
	_ = make(T, 0) //@ diag(`should use make(T) instead`)
}

func fn4[T any, C chan T]() {
	_ = make(chan T, 0) //@ diag(`should use make(chan T) instead`)
	_ = make(C, 0)      //@ diag(`should use make(C) instead`)
}

func fn5[T []int]() {
	_ = make(T, 0) // don't flag this, T isn't a channel
}

type I interface {
	chan int
}

func fn6[T I]() {
	_ = make(T, 0) //@ diag(`should use make(T) instead`)
}
