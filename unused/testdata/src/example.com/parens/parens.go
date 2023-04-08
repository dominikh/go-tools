package p

func F(c chan bool) { //@ used("F", true), used("c", true)
	select {
	case (<-c):
	case _ = (<-c):
	}
}
