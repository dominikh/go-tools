package pkg

func fn2() {
	var ch chan int
	<-ch
	_ = <-ch //@ diag(`unnecessary assignment to the blank identifier`)
	select {
	case <-ch:
	case _ = <-ch: //@ diag(`unnecessary assignment to the blank identifier`)
	}
	x := <-ch
	y, _ := <-ch, <-ch
	_, z := <-ch, <-ch
	_, _, _ = x, y, z
}
