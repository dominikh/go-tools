package pkg

func fn(b1, b2 bool) {
	if !!b1 { // MATCH /negating a boolean twice/
	}

	if b1 && !!b2 { // MATCH /negating a boolean twice/
	}

	if !(!b1) { // doesn't match, maybe it should
	}

	if !b1 {
	}

	if !b1 && !b2 {
	}
}
