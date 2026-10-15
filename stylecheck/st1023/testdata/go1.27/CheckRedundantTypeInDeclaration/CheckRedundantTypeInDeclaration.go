package pkg

type T struct{}

func (T) M[P any](v P) P { return v }

func F(t T) string {
	var f func(string) string = t.M
	return f("a")
}
