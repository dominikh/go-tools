//go:build go1.18

package pkg

func _[T int | string](x []T) {
	if x != nil { // want `unnecessary nil check around range`
		for range x {
		}
	}
}

func _[T int | string, S []T](x S) {
	if x != nil { // want `unnecessary nil check around range`
		for range x {
		}
	}
}

func _[T []string](x T) {
	if x != nil { // want `unnecessary nil check around range`
		for range x {
		}
	}
}

func _[T chan int](x T) {
	if x != nil {
		for range x {
		}
	}
}

func _[T any, S chan T](x S) {
	if x != nil {
		for range x {
		}
	}
}
