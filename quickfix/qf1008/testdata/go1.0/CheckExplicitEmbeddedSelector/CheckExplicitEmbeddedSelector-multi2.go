package pkg

type Embedded2 struct{}
type Embedded struct{ Embedded2 }
type Wrapper struct{ Embedded }

func (e Embedded2) String() string { return "" }
func (w Wrapper) String() string {
	// Either Embedded or Embedded2 can be removed, but removing both would
	// change the semantics
	return w.Embedded.Embedded2.String() //@ diag(`could remove embedded field "Embedded"`), diag(`could remove embedded field "Embedded2`)
}
