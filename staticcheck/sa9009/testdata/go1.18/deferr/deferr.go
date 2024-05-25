package deferr

func x() {

	defer tpReturnFuncInt[int](0) //@ diag(`defered return function not called`)
	defer tpReturnFuncInt(0)(0)
}

func tpReturnFuncInt[T any](T) func(int) int { return func(int) int { return 0 } }
