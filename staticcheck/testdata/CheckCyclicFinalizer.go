package pkg

import (
	"fmt"
	"runtime"
)

func fn() {
	var x *int
	foo := func(y *int) { fmt.Println(x) }
	if true {
		foo = func(y *int) { fmt.Println(x) }
	}
	if false {
		foo = nil
	}
	runtime.SetFinalizer(x, foo)
	runtime.SetFinalizer(x, func(_ *int) {
		fmt.Println(x)
	})

	foo = func(y *int) { fmt.Println(y) }
	runtime.SetFinalizer(x, foo)
	runtime.SetFinalizer(x, func(y *int) {
		fmt.Println(y)
	})
}

// MATCH:17 /the finalizer closes over the object, preventing the finalizer from ever running \(at .+:10:9/
// MATCH:17 /the finalizer closes over the object, preventing the finalizer from ever running \(at .+:12:9/
// MATCH:18 /the finalizer closes over the object, preventing the finalizer from ever running \(at .+:18:26/
