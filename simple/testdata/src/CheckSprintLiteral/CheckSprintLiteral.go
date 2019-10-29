package pkg

import "fmt"

func fn() {
	fmt.Sprint("foo")  // want `unnecessary use of fmt\.Sprint`
	fmt.Sprintf("foo") // want `unnecessary use of fmt\.Sprintf`
	fmt.Sprintf("foo %d")
	fmt.Sprintf("foo %d", 1)

	var x string
	fmt.Sprint(x)
}
