package pkg

import _ "CheckDeprecatedassist"          //@ diag(`Alas, it is deprecated.`)
import _ "AnotherCheckDeprecatedassist"   //@ diag(`Alas, it is deprecated.`)
import foo "AnotherCheckDeprecatedassist" //@ diag(`Alas, it is deprecated.`)
import "AnotherCheckDeprecatedassist"     //@ diag(`Alas, it is deprecated.`)

func init() {
	foo.Fn()
	AnotherCheckDeprecatedassist.Fn()
}
