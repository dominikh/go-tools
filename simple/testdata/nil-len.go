package pkg

func fn() {
	var pa *[5]int
	var s []int
	var m map[int]int
	var ch chan int

	if s == nil || len(s) == 0 { // MATCH /should omit nil check/
	}
	if m == nil || len(m) == 0 { // MATCH /should omit nil check/
	}
	if ch == nil || len(ch) == 0 { // MATCH /should omit nil check/
	}

	if s != nil && len(s) != 0 { // MATCH /should omit nil check/
	}
	if m != nil && len(m) > 0 { // MATCH /should omit nil check/
	}
	if ch != nil && len(ch) == 5 { // MATCH /should omit nil check/
	}

	if pa == nil || len(pa) == 0 { // nil check cannot be removed with pointer to an array
	}
	if s == nil || len(m) == 0 { // different variables
	}
	if s != nil && len(m) == 1 { // different variables
	}

	var ch2 chan int
	if ch == ch2 || len(ch) == 0 { // not comparing with nil
	}
	if ch != ch2 && len(ch) != 0 { // not comparing with nil
	}

	if s != nil && len(s) == 0 { // nil check is not redundant here
	}
}
