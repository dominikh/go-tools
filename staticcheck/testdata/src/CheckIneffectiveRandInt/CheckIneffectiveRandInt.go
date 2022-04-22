package pkg

import "math/rand"

func fn() {
	type T struct {
		rng rand.Rand
	}

	_ = rand.Intn(1)   //@ diag(re`rand\.Intn\(n\) generates.+rand\.Intn\(1\) therefore`)
	_ = rand.Int63n(1) //@ diag(re`rand\.Int63n\(n\) generates.+rand\.Int63n\(1\) therefore`)
	var t T
	_ = t.rng.Intn(1) //@ diag(re`\(\*math/rand\.Rand\)\.Intn\(n\) generates.+t\.rng\.Intn\(1\) therefore`)

	_ = rand.Intn(2)
}
