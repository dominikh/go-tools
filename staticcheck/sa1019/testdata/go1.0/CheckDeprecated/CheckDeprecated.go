// Deprecated: this package is deprecated.
package pkg

import _ "example.com/CheckDeprecated.assist"          //@ diag(`Alas, it is deprecated.`)
import _ "example.com/AnotherCheckDeprecated.assist"   //@ diag(`Alas, it is deprecated.`)
import foo "example.com/AnotherCheckDeprecated.assist" //@ diag(`Alas, it is deprecated.`)
import "example.com/AnotherCheckDeprecated.assist"     //@ diag(`Alas, it is deprecated.`)
import ae "example.com/CheckDeprecated.assist_external"

func init() {
	foo.Fn()
	AnotherCheckDeprecatedassist.Fn()

	// Field is deprecated, but we're using it from the same package, which is fine.
	var s S
	_ = s.Field

	s2 := ae.SD{
		D: "used", //@ diag(`external don't use me`)
	}
	_ = s2
	// Struct with the same name should not be flagged
	s3 := ae.SN{
		D:"also",
	}
	_ = s3
	// Other Key-Value expressions should be safely ignored
	_ = map[string]string {
		"left":"right",
	}
}


type S struct {
	 // Deprecated: this is deprecated.
	 Field int
}
