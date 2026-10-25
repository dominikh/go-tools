package pkg

import ctx "context"

func fn(c ctx.Context) {}

func call() {
	ctx := 1
	_ = ctx
	fn(nil) //@ diag(`do not pass a nil Context`)
}
