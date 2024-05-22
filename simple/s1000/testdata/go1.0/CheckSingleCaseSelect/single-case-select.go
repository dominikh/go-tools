package pkg

func fn() {
	var ch chan int
	select { //@ diag(`should use a simple channel send`)
	case <-ch:
	}
outer:
	for { //@ diag(`should use for range`)
		select {
		case <-ch:
			break outer
		}
	}

	for { //@ diag(`should use for range`)
		select {
		case x := <-ch:
			_ = x
		}
	}

	for {
		select { //@ diag(`should use a simple channel send`)
		case ch <- 0:
		}
	}
}
