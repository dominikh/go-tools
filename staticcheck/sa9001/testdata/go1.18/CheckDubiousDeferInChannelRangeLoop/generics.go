package pkg

func tpfn[T chan int]() {
	var ch T
	for range ch {
		defer println() //@ diag(`defers in this range loop`)
	}
}
