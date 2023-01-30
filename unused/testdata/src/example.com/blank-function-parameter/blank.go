package pkg

type customType int //@ used("customType", true)

func Foo(customType) {} //@ used("Foo", true), used("", true)
func bar(customType) {} //@ used("bar", false), quiet("")
