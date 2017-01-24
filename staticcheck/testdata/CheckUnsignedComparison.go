package pkg

func fn(x uint32) {
	if x >= 0 { // MATCH /unsigned values are always >= 0/
	}
	if x < 0 { // MATCH /unsigned values are never < 0/
	}
	if x <= 0 { // MATCH /'x <= 0' for unsigned values of x is the same as 'x == 0'/
	}
}
