package pkg

import "math/rand/v2"

func fn() {
	type T struct {
		rng rand.Rand
	}

	_ = rand.IntN(1)   //@ diag(re`rand/v2\.IntN\(n\) generates.+rand\.IntN\(1\) therefore`)
	_ = rand.Int64N(1) //@ diag(re`rand/v2\.Int64N\(n\) generates.+rand\.Int64N\(1\) therefore`)
	_ = rand.N(1)      //@ diag(re`rand/v2\.N\(n\) generates.+rand\.N\(1\) therefore`)
	var t T
	_ = t.rng.IntN(1) //@ diag(re`\(\*math/rand/v2\.Rand\)\.IntN\(n\) generates.+t\.rng\.IntN\(1\) therefore`)

	_ = rand.IntN(2)
}
