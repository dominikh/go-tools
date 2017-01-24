package pkg

func fn1(b1, b2 bool) {
	if b1 && !b2 {
	} else if b1 {
	} else if b1 && !b2 { // MATCH /condition occurs multiple times/
	} else if b1 { // MATCH /condition occurs multiple times/
	} else {
	}
}

func fn2(b1, b2 bool, ch chan string) {
	if b1 && !b2 {
	} else if b1 {
	} else if <-ch == "" {
	} else if <-ch == "" {
	} else {
	}
}

func fn3() {
	if gen() {
		println()
	} else if gen() {
		println()
	}
}

func fn4() {
	if s := gen2(); s == "" {
	} else if s := gen2(); s == "" {
	}
}

func fn5() {
	var s string
	if s = gen2(); s == "" {
	} else if s != "foo" {
	} else if s = gen2(); s == "" {
	} else if s != "foo" {
	}
}

func gen() bool    { return false }
func gen2() string { return "" }
