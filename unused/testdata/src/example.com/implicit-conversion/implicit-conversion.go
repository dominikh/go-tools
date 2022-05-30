package pkg

// https://staticcheck.io/issues/810

type Thing struct { //@ used("Thing", true)
	has struct { //@ used("has", true)
		a bool //@ used("a", true)
	}
}

func Fn() { //@ used("Fn", true)
	type temp struct { //@ used("temp", true)
		a bool //@ used("a", true)
	}

	x := Thing{ //@ used("x", true)
		has: temp{true},
	}
	_ = x
}
