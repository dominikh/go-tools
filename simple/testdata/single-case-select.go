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
}
