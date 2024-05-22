package pkg

func done() bool { return false }

var a, b int
var x bool

func fn() {
	for {
		if done() { //@ diag(`could lift into loop condition`)
			break
		}
	}

	for {
		if !done() { //@ diag(`could lift into loop condition`)
			break
		}
	}

	for {
		if a > b || b > a { //@ diag(`could lift into loop condition`)
			break
		}
	}

	for {
		if x && (a == b) { //@ diag(`could lift into loop condition`)
			break
		}
	}

	for {
		if done() { //@ diag(`could lift into loop condition`)
			break
		}
		println()
	}

	for {
		println()
		if done() {
			break
		}
	}

	for {
		if done() {
			println()
			break
		}
	}
}
