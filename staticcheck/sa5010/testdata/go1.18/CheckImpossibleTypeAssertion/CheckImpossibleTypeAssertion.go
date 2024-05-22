package pkg

type ExampleType[T uint32 | uint64] interface {
	SomeMethod() T
}

func Fn1[T uint32 | uint64]() {
	var iface ExampleType[uint32]
	_ = iface.(ExampleType[T])
}

func Fn2[T uint64]() {
	// In theory we should flag this, but we don't.
	var iface ExampleType[uint32]
	_ = iface.(ExampleType[T])
}
