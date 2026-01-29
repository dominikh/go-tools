package pkg

// Uncomparable types that implement error

type ErrWithSlice struct {
	Details []string
}

func (e ErrWithSlice) Error() string { return "" }

type ErrWithMap struct {
	Data map[string]int
}

func (e ErrWithMap) Error() string { return "" }

type ErrWithFunc struct {
	Handler func()
}

func (e ErrWithFunc) Error() string { return "" }

// Embedded uncomparable

type ErrEmbedded struct {
	ErrWithSlice
}

func (e ErrEmbedded) Error() string { return "" }

// Nested uncomparable

type Inner struct {
	Values []int
}

type ErrNested struct {
	Inner Inner
}

func (e ErrNested) Error() string { return "" }

// Comparable types

type ErrComparable struct {
	Code int
	Msg  string
}

func (e ErrComparable) Error() string { return "" }

type ErrEmpty struct{}

func (e ErrEmpty) Error() string { return "" }

type ErrWithInterface struct {
	Err error
}

func (e ErrWithInterface) Error() string { return "" }

type ErrArray struct {
	Codes [3]int
}

func (e ErrArray) Error() string { return "" }

// Non-error interface

type UncomparableStringer struct {
	Data []string
}

func (u UncomparableStringer) String() string { return "" }

func fn() {
	// Should warn: uncomparable types converted to error
	_ = error(ErrWithSlice{})   //@ diag(`conversion of uncomparable type`)
	_ = error(ErrWithMap{})     //@ diag(`conversion of uncomparable type`)
	_ = error(ErrWithFunc{})    //@ diag(`conversion of uncomparable type`)
	_ = error(ErrEmbedded{})    //@ diag(`conversion of uncomparable type`)
	_ = error(ErrNested{})      //@ diag(`conversion of uncomparable type`)

	// Should NOT warn: pointer conversions
	_ = error(&ErrWithSlice{})
	_ = error(&ErrWithMap{})
	_ = error(&ErrWithFunc{})

	// Should NOT warn: comparable types
	_ = error(ErrComparable{})
	_ = error(ErrEmpty{})
	_ = error(ErrWithInterface{})
	_ = error(ErrArray{})

	// Should NOT warn: non-error interface
	var _ interface{ String() string } = UncomparableStringer{}
}

func fnReturn() error {
	return ErrWithSlice{} //@ diag(`conversion of uncomparable type`)
}

func fnAssign() {
	var err error
	err = ErrWithSlice{} //@ diag(`conversion of uncomparable type`)
	_ = err
}
