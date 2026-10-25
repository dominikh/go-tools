package pkg

type Floater interface {
	~float64
}

func generic[T Floater](a, b T) bool {
	return !(a < b)
}
