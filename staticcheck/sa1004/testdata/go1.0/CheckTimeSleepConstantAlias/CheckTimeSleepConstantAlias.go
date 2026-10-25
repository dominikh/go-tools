package pkg

import timelib "time"

const c1 = 1
const c2 = 200

func fn() {
	timelib.Sleep(1)  //@ diag(`sleeping for 1`)
	timelib.Sleep(42) //@ diag(`sleeping for 42`)
	timelib.Sleep(201)
	timelib.Sleep(c1)
	timelib.Sleep(c2)
	timelib.Sleep(2 * timelib.Nanosecond)
	timelib.Sleep(timelib.Nanosecond)
}
