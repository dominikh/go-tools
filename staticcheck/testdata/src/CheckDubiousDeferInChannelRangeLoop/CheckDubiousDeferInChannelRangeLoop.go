package pkg

func fn() {
	var ch chan int
	for range ch {
		defer println() //@ diag(`defers in this range loop`)
	}
}
