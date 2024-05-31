package pkg

import "context"

type T string

type T2 struct {
	A int
}

type T3 struct {
	A []int
}

func fn(arg1 interface{}, arg2 string) {
	var ctx context.Context
	context.WithValue(ctx, "hi", nil) //@ diag(`should not use built-in type string`)
	context.WithValue(ctx, arg1, nil)
	context.WithValue(ctx, arg2, nil) //@ diag(`should not use built-in type string`)
	v1 := interface{}("byte")
	context.WithValue(ctx, v1, nil) //@ diag(`should not use built-in type string`)

	var key T
	context.WithValue(ctx, key, nil)
	v2 := interface{}(key)
	context.WithValue(ctx, v2, nil)
	context.WithValue(ctx, T(""), nil)
	context.WithValue(ctx, string(key), nil) //@ diag(`should not use built-in type string`)

	context.WithValue(ctx, []byte(nil), nil) //@ diag(`must be comparable`)
	context.WithValue(ctx, T2{}, nil)
	context.WithValue(ctx, T3{}, nil) //@ diag(`must be comparable`)

	context.WithValue(ctx, struct{ key string }{"k"}, nil)
	var empty struct{}
	context.WithValue(ctx, struct{}{}, nil) //@ diag(`should not use empty anonymous struct`)
	context.WithValue(ctx, empty, nil)      //@ diag(`should not use empty anonymous struct`)
}
