package pkg

// https://staticcheck.io/issues/1333

func normal1()     {}           //@ used("normal1", false)
func normal2()     {}           //@ used("normal2", true)
func normal3() int { return 0 } //@ used("normal3", false)
func normal4() int { return 0 } //@ used("normal4", true)
