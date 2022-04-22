//go:build go1.18

package pkg

import "encoding/json"

type LMap[K comparable, V any] struct {
	M1 map[K]V
	M2 map[K]chan int
}

func (lm *LMap[K, V]) MarshalJSON() {
	json.Marshal(lm.M1)
	json.Marshal(lm.M2) // want `unsupported type`
}
