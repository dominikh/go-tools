package pkg

func _[T map[int]int]() {
	for range (T)(nil) {
		if true {
			println()
		}
		break
	}
}

func _[K comparable, V any, M ~map[K]V]() {
	for range (M)(nil) {
		if true {
			println()
		}
		break
	}
}

func _[T []int]() {
	for range (T)(nil) {
		if true {
			println()
		}
		break //@ diag(`the surrounding loop is unconditionally terminated`)
	}
}

func _[T any, S ~[]T](x S) {
	for range x {
		if true {
			println()
		}
		break //@ diag(`the surrounding loop is unconditionally terminated`)
	}
}
