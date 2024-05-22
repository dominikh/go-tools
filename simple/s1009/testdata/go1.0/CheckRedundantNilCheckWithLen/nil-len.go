package pkg

func fn() {
	var pa *[5]int
	var s []int
	var m map[int]int
	var ch chan int

	if s == nil || len(s) == 0 { //@ diag(`should omit nil check`)
	}
	if m == nil || len(m) == 0 { //@ diag(`should omit nil check`)
	}
	if ch == nil || len(ch) == 0 { //@ diag(`should omit nil check`)
	}

	if s != nil && len(s) != 0 { //@ diag(`should omit nil check`)
	}
	if m != nil && len(m) > 0 { //@ diag(`should omit nil check`)
	}
	if s != nil && len(s) > 5 { //@ diag(`should omit nil check`)
	}
	if s != nil && len(s) >= 5 { //@ diag(`should omit nil check`)
	}
	const five = 5
	if s != nil && len(s) == five { //@ diag(`should omit nil check`)
	}

	if ch != nil && len(ch) == 5 { //@ diag(`should omit nil check`)
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

	const zero = 0
	if s != nil && len(s) == zero { // nil check is not redundant here
	}
	if s != nil && len(s) == 0 { // nil check is not redundant here
	}
	if s != nil && len(s) >= 0 { // nil check is not redundant here (though len(s) >= 0 is)
	}
	one := 1
	if s != nil && len(s) == one { // nil check is not redundant here
	}
	if s != nil && len(s) == len(m) { // nil check is not redundant here
	}
	if s != nil && len(s) != 1 { // nil check is not redundant here
	}
	if s != nil && len(s) < 5 { // nil check is not redundant here
	}
	if s != nil && len(s) <= 5 { // nil check is not redundant here
	}
	if s != nil && len(s) != len(ch) { // nil check is not redundant here
	}
}

func fn3() {
	var x []int

	if x == nil || len(x) == 0 { //@ diag(`should omit nil check`)
	}

	len := func([]int) int { return 10 }
	if x == nil || len(x) == 0 {
	}
}
