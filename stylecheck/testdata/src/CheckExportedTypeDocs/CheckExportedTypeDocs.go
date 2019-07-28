package pkg

// Some type
type t1 struct{}

// Some type // want `comment on exported type`
type T2 struct{}

// T3 is amazing
type T3 struct{}

type (
	// Some type // want `comment on exported type`
	T4 struct{}
	// The T5 type is amazing
	T5 struct{}
	// Some type
	t6 struct{}
)

// Some types
type (
	T7 struct{}
	T8 struct{}
)

// Some types
type (
	T9 struct{}
)

func fn() {
	// Some type
	type T1 struct{}
}
