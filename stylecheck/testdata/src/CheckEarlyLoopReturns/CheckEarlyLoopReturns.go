package pkg

import (
	"math/rand"
	"fmt"
)

func fn() {
	n := rand.Intn(10)
	for { //@ diag(`for loop with a single if statement in the body is a candidate for early return`)
		if n > 5 {
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
		}
	}

	// Comments should not affect the warning.
	for { //@ diag(`for loop with a single if statement in the body is a candidate for early return`)
		// Just an uninteresting comment.
		if n > 5 {
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
		}
	}

	// The if statement has < 10 lines so no warning should be shown.
	for {
		if n > 5 {
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
		}
	}
	for {
		// This if condition should NOT be caught because there is another statement after
		// the if condition within the for loop.
		if n > 5 {
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
		}
		fmt.Println("Extra line")
	}
	for {
		fmt.Println("Extra line")
		// This if condition should NOT be caught because there is another statement before
		// the if condition within the for loop.
		if n > 5 {
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
			fmt.Println("1")
		}
	}

}
