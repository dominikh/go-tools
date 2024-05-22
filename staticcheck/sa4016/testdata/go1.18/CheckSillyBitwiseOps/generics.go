package pkg

func tpfn[T int](x T) {
	_ = x & 0 //@ diag(`always equals 0`)
}
