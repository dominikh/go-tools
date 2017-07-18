package pkg

import (
	"time"
)

func fn1() {
	var t1, t2 time.Time

	_ = t1 == t2 // MATCH /using == or != operators to compare time is problematic/
	_ = t1 != t2 // MATCH /using == or != operators to compare time is problematic/
}

func fn2() {
	var t1, t2 *time.Time

	_ = *t1 == *t2 // MATCH /using == or != operators to compare time is problematic/
	_ = *t1 != *t2 // MATCH /using == or != operators to compare time is problematic/
}

func fn3() {
	var t1 time.Time
	var t2 *time.Time

	_ = t1 == *t2 // MATCH /using == or != operators to compare time is problematic/
	_ = t1 != *t2 // MATCH /using == or != operators to compare time is problematic/
}
