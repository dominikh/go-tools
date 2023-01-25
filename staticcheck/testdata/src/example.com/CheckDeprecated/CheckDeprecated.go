package pkg

import _ "example.com/CheckDeprecated.assist"          //@ diag(`Alas, it is deprecated.`)
import _ "example.com/AnotherCheckDeprecated.assist"   //@ diag(`Alas, it is deprecated.`)
import foo "example.com/AnotherCheckDeprecated.assist" //@ diag(`Alas, it is deprecated.`)
import "example.com/AnotherCheckDeprecated.assist"     //@ diag(`Alas, it is deprecated.`)

func init() {
	foo.Fn()
	AnotherCheckDeprecatedassist.Fn()
}
