package pkg

func do() bool {
	type Info struct {
		idx int
	}

	var state map[Info]int
	// Don't crash on KeyValueExpr
	return !(state[Info{idx: 6}] == 6 || false) //@ diag(`could apply De Morgan's law`)
}
