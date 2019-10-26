package pkg

func foo(a, b int) int { return a + b } // want foo:"is pure"
func bar(a, b int) int {
	println(a + b)
	return a + b
}

func empty()            {}
func stubPointer() *int { return nil } // want stubPointer:"is pure"
func stubInt() int      { return 0 }   // want stubInt:"is pure"

func fn3() {
	empty()
	stubPointer()
	stubInt()
}
