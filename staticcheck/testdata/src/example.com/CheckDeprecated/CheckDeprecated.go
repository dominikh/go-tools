// Deprecated: this package is deprecated.
package pkg

import _ "example.com/CheckDeprecated.assist"          //@ diag(`Alas, it is deprecated.`)
import _ "example.com/AnotherCheckDeprecated.assist"   //@ diag(`Alas, it is deprecated.`)
import foo "example.com/AnotherCheckDeprecated.assist" //@ diag(`Alas, it is deprecated.`)
import "example.com/AnotherCheckDeprecated.assist"     //@ diag(`Alas, it is deprecated.`)

func init() {
	foo.Fn()
	AnotherCheckDeprecatedassist.Fn()

	// Field is deprecated, but we're using it from the same package, which is fine.
	var s S
	_ = s.Field
}


type S struct {
	 // Deprecated: this is deprecated.
	 Field int
}
