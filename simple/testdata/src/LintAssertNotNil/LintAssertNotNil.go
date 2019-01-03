package pkg

func fn(i interface{}, x interface{}) {
	if _, ok := i.(string); ok && i != nil { // MATCH "when ok is true, i can't be nil"
	}
	if _, ok := i.(string); i != nil && ok { // MATCH "when ok is true, i can't be nil"
	}
	if _, ok := i.(string); i != nil || ok {
	}
	if _, ok := i.(string); i != nil && !ok {
	}
	if _, ok := i.(string); i == nil && ok {
	}
	if i != nil {
		if _, ok := i.(string); ok { // MATCH "when ok is true, i can't be nil"
		}
	}
	if i != nil {
		if _, ok := i.(string); ok {
		}
		println(i)
	}
	if i == nil {
		if _, ok := i.(string); ok {
		}
	}
	if i != nil {
		if _, ok := i.(string); !ok {
		}
	}
	if x != nil {
		if _, ok := i.(string); ok {
		}
	}
	if i != nil {
		if _, ok := x.(string); ok {
		}
	}
}
