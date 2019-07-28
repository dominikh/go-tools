package pkg

// whatever
func foo() {}

// Foo is amazing
func Foo() {}

// Whatever // want `comment on exported function`
func Bar() {}

type T struct{}

// Whatever
func (T) foo() {}

// Foo is amazing
func (T) Foo() {}

// Whatever // want `comment on exported method`
func (T) Bar() {}
