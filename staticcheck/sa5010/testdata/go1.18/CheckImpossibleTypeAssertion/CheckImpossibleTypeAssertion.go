package pkg

type ExampleType[T uint32 | uint64] interface {
	SomeMethod() T
}

func Fn1[T uint32 | uint64]() {
	var iface ExampleType[uint32]
	_ = iface.(ExampleType[T])
}

func Fn2[T uint64]() {
	var iface ExampleType[uint32]
	// TODO(dh): once we support generics, flag this
	_ = iface.(ExampleType[T])
}

type I1[E any] interface {
	Do(E)
	Moo(E)
}

type I2[E any] interface {
	Do(E)
	Moo(E)
}

type I3[E any] interface {
	Do(E)
	Moo()
}

func New[T any]() {
	var x I1[T]
	_ = x.(I2[T])

	// TODO(dh): once we support generics, flag this
	_ = x.(I3[T])
}
