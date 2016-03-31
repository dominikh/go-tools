package pkg

func fn() {
	var b1, b2 []byte
	for i, v := range b1 { // MATCH /should use copy/
		b2[i] = v
	}
}
