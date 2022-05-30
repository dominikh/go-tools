package pkg

//go:cgo_export_dynamic
func foo() {} //@ used("foo", true)

func bar() {} //@ used("bar", false)
