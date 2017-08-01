package pkg

func fn() {
	str := []string{}

	if str != nil { // MATCH /unnecessary nil check around range/
		for _, s := range str {
			s = s + "A"
		}
	}
}
