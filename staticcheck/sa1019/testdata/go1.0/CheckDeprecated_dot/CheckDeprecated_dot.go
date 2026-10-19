package pkg

import . "example.com/CheckDeprecated_dot.assist"

func fn() {
	OldFunc() //@ diag(`is deprecated: Use NewFunc instead.`)
	NewFunc()
	_ = OldVar //@ diag(`is deprecated: Use NewVar instead.`)
	_ = NewVar
}
