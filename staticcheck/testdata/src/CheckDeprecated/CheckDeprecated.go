package pkg

import _ "CheckDeprecated.assist"          //@ diag(`Alas, it is deprecated.`)
import _ "AnotherCheckDeprecated.assist"   //@ diag(`Alas, it is deprecated.`)
import foo "AnotherCheckDeprecated.assist" //@ diag(`Alas, it is deprecated.`)
import "AnotherCheckDeprecated.assist"     //@ diag(`Alas, it is deprecated.`)

func init() {
	foo.Fn()
	AnotherCheckDeprecatedassist.Fn()
}
