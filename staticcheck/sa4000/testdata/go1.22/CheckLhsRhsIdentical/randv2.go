package pkg

import "math/rand/v2"

func fn() {
	_ = rand.N[uint](5) == rand.N[uint](5)
}
