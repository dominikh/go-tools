package pkg

// https://staticcheck.io/issues/507

var x = [3]int{1, 2, 3} //@ used("x", false)
