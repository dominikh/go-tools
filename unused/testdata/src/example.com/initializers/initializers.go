package pkg

// https://staticcheck.dev/issues/507

var x = [3]int{1, 2, 3} //@ used("x", false)
