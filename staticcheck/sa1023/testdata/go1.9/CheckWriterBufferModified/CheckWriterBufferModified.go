package pkg

type T5 struct{}
type T6 struct{}

type AliasByte = byte
type AliasByteSlice = []byte
type AliasInt = int
type AliasError = error

func (T5) Write(b []AliasByte) (int, error) {
	b[0] = 0 //@ diag(`io.Writer.Write must not modify the provided buffer`)
	return 0, nil
}

func (T6) Write(b AliasByteSlice) (AliasInt, AliasError) {
	b[0] = 0 //@ diag(`io.Writer.Write must not modify the provided buffer`)
	return 0, nil
}
