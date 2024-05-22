package pkg

import "fmt"

type MyByte byte
type T1 []MyByte

func fn() {
	var t1 T1
	_ = fmt.Sprintf("%s", t1) //@ diag(`underlying type is a slice of bytes`)
}
