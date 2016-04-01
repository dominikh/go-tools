package pkg

func fn() {
	var ch chan int
	select { // MATCH /should use a simple channel send/
	case <-ch:
	}
outer:
	for { // MATCH /should use for range/
		select {
		case <-ch:
			break outer
		}
	}

	for { // MATCH /should use for range/
		select {
		case _ = <-ch:
		}
	}

	for {
		select { // MATCH /should use a simple channel send/
		case ch <- 0:
		}
	}
}
