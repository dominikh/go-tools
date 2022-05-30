package pkg

func Fn1() { //@ used("Fn1", true)
	type Foo[T any] struct { //@ used("Foo", true), used("T", true)
		Id   int `json:"id"`   //@ used("Id", true)
		Data T   `json:"data"` //@ used("Data", true)
	}
	type Bar struct { //@ used("Bar", true)
		X int `json:"x"` //@ used("X", true)
		Y int `json:"y"` //@ used("Y", true)
	}
	v := Foo[[]Bar]{} //@ used("v", true)
	_ = v
}

func Fn2() { //@ used("Fn2", true)
	type Foo[T any] struct{} //@ used("Foo", true), used("T", true)
	type Bar struct{}        //@ used("Bar", true)
	v := Foo[[]Bar]{}        //@ used("v", true)
	_ = v                    // just use it, but could be some json.Unmarshal, for instance
}
