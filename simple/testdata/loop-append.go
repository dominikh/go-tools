package pkg

func fn() {
	var x []interface{}
	var y []int

	for _, v := range y {
		x = append(x, v)
	}

	var a, b []int
	for _, v := range a { // MATCH /should replace loop/
		b = append(b, v)
	}

	var m map[string]int
	var c []int
	for _, v := range m {
		c = append(c, v)
	}
}
