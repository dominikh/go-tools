package pkg

import "context"

func tpfn1[T any](ctx context.Context, x T)             {}
func tpfn2[T1, T2 any](ctx context.Context, x T1, y T2) {}

func tpbar() {
	tpfn1[int](nil, 0) //@ diag(`do not pass a nil Context`)
	tpfn1(nil, 0)      //@ diag(`do not pass a nil Context`)

	tpfn2[int, int](nil, 0, 0) //@ diag(`do not pass a nil Context`)
	tpfn2(nil, 0, 0)           //@ diag(`do not pass a nil Context`)
}
