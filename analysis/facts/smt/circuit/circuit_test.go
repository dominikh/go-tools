package circuit

import (
	"fmt"
	"testing"
)

func TestFoo(t *testing.T) {
	a := Var(0)

	q := And(a, Not(a))

	fmt.Println(q)
}
