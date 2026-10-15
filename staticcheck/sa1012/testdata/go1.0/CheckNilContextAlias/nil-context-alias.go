package pkg

import ctx "context"

func fn(c ctx.Context) {}

func call() {
	fn(nil) //@ diag(`do not pass a nil Context`)
}
