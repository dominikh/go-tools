//lint:file-ignore U1000 consider everything in here used

package pkg

type t9 struct{} //@ used("t9", true)

func (t9) fn1() {} //@ used("fn1", true)
